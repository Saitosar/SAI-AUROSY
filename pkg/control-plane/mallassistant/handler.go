package mallassistant

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sai-aurosy/platform/internal/mall"
	"github.com/sai-aurosy/platform/pkg/control-plane/cognitive"
	"github.com/sai-aurosy/platform/pkg/control-plane/events"
	"github.com/sai-aurosy/platform/pkg/control-plane/tasks"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

const (
	greetingText      = "Hello! Welcome to the mall. How can I help you?"
	followMeTemplate  = "Follow me. I will guide you to %s."
	arrivalTemplate   = "We have arrived. %s is here."
	storeNotFoundText = "I'm sorry, I couldn't find that store. Could you try another?"
	defaultLanguage   = "en"
	navigateTaskWait  = 30 * time.Second
	standbyCoordinates = "0,0,0"
)

// VisitorRequestRegistry routes visitor requests to active mall assistant handlers.
type VisitorRequestRegistry struct {
	mu       sync.RWMutex
	channels map[string]chan string
}

// NewVisitorRequestRegistry creates a new registry.
func NewVisitorRequestRegistry() *VisitorRequestRegistry {
	return &VisitorRequestRegistry{
		channels: make(map[string]chan string),
	}
}

// Register adds a channel for the robot. Returns the channel the handler should receive on.
func (r *VisitorRequestRegistry) Register(robotID string) chan string {
	ch := make(chan string, 4)
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.channels[robotID]; ok {
		close(old)
	}
	r.channels[robotID] = ch
	return ch
}

// Unregister removes the channel for the robot.
func (r *VisitorRequestRegistry) Unregister(robotID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ch, ok := r.channels[robotID]; ok {
		close(ch)
		delete(r.channels, robotID)
	}
}

// Submit sends a visitor request to the handler for the robot. Returns false if no active handler.
func (r *VisitorRequestRegistry) Submit(robotID string, text string) bool {
	r.mu.RLock()
	ch, ok := r.channels[robotID]
	r.mu.RUnlock()
	if !ok || ch == nil {
		return false
	}
	select {
	case ch <- text:
		return true
	default:
		return false
	}
}

// Handler runs the mall assistant scenario.
type Handler struct {
	bus              *telemetry.Bus
	cognitiveGateway cognitive.Gateway
	taskStore        tasks.Store
	eventBroadcaster *events.Broadcaster
	requestRegistry  *VisitorRequestRegistry
	mallService      *mall.Service
	onTaskCompleted  func(taskID, robotID, status string)
}

// HandlerConfig configures the handler.
type HandlerConfig struct {
	MallService     *mall.Service
	OnTaskCompleted func(taskID, robotID, status string)
}

// NewHandler creates a new mall assistant handler.
func NewHandler(
	bus *telemetry.Bus,
	cognitiveGateway cognitive.Gateway,
	taskStore tasks.Store,
	eventBroadcaster *events.Broadcaster,
	requestRegistry *VisitorRequestRegistry,
	cfg HandlerConfig,
) *Handler {
	return &Handler{
		bus:              bus,
		cognitiveGateway: cognitiveGateway,
		taskStore:        taskStore,
		eventBroadcaster: eventBroadcaster,
		requestRegistry:  requestRegistry,
		mallService:      cfg.MallService,
		onTaskCompleted:  cfg.OnTaskCompleted,
	}
}

// Run executes the mall assistant scenario. Blocks until ctx is done or scenario completes.
func (h *Handler) Run(ctx context.Context, taskID, robotID, tenantID, operatorID string) {
	defer h.requestRegistry.Unregister(robotID)

	if h.eventBroadcaster != nil {
		h.eventBroadcaster.Broadcast("visitor_interaction_started", map[string]any{
			"task_id":  taskID,
			"robot_id": robotID,
		})
	}

	if err := h.speak(ctx, robotID, greetingText); err != nil {
		slog.Warn("mall_assistant speak greeting failed", "robot_id", robotID, "error", err)
	}

	requestCh := h.requestRegistry.Register(robotID)

	var visitorText string
	select {
	case <-ctx.Done():
		h.completeTask(taskID, robotID, tasks.StatusCancelled)
		return
	case t, ok := <-requestCh:
		if !ok {
			h.completeTask(taskID, robotID, tasks.StatusCancelled)
			return
		}
		visitorText = t
	}

	store, err := h.processRequest(ctx, robotID, visitorText)
	if err != nil || store == nil {
		if err != nil {
			slog.Warn("mall_assistant process request failed", "robot_id", robotID, "error", err)
		}
		_ = h.speak(ctx, robotID, storeNotFoundText)
		h.completeTask(taskID, robotID, tasks.StatusCompleted)
		return
	}

	const mallID = "default"
	var destNode *mall.NavNode
	var baseNode *mall.NavNode
	var route []mall.NavNode
	var estimatedDistance float64
	var routeNodeIDs []string

	if h.mallService != nil {
		if n, err := h.mallService.FindStoreNode(ctx, mallID, store.Name); err == nil {
			destNode = n
			if h.eventBroadcaster != nil {
				h.eventBroadcaster.Broadcast("mall_store_resolved", map[string]any{
					"mall_id":    mallID,
					"store_name": store.Name,
					"node_id":   n.ID,
				})
			}
		}
		if destNode != nil {
			if b, err := h.mallService.GetBasePoint(ctx, mallID); err == nil {
				baseNode = b
			}
		}
		if destNode != nil && baseNode != nil {
			if r, dist, err := h.mallService.CalculateRoute(ctx, mallID, baseNode.ID, destNode.ID); err == nil {
				route = r
				estimatedDistance = dist
				for _, n := range r {
					routeNodeIDs = append(routeNodeIDs, n.ID)
				}
				if h.eventBroadcaster != nil {
					h.eventBroadcaster.Broadcast("mall_route_calculated", map[string]any{
						"mall_id":             mallID,
						"from_node":           baseNode.ID,
						"to_node":             destNode.ID,
						"route_length":        len(route),
						"estimated_distance":  dist,
					})
				}
			}
		}
	}

	if h.eventBroadcaster != nil {
		h.eventBroadcaster.Broadcast("navigation_started", map[string]any{
			"task_id":  taskID,
			"robot_id": robotID,
			"store":    store.Name,
		})
	}

	followText := fmt.Sprintf(followMeTemplate, store.Name)
	if err := h.speak(ctx, robotID, followText); err != nil {
		slog.Warn("mall_assistant speak follow failed", "robot_id", robotID, "error", err)
	}

	navTaskID, err := h.createNavigateTask(robotID, tenantID, operatorID, store, destNode, routeNodeIDs, estimatedDistance)
	if err != nil {
		slog.Error("mall_assistant create navigate task failed", "robot_id", robotID, "error", err)
		h.completeTask(taskID, robotID, tasks.StatusFailed)
		return
	}

	if h.eventBroadcaster != nil && destNode != nil && baseNode != nil {
		h.eventBroadcaster.Broadcast("mall_navigation_requested", map[string]any{
			"mall_id":             mallID,
			"robot_id":            robotID,
			"store_name":          store.Name,
			"from_node":           baseNode.ID,
			"to_node":             destNode.ID,
			"route_length":       len(routeNodeIDs),
			"estimated_distance": estimatedDistance,
		})
	}

	h.waitForNavigation(ctx, navTaskID, robotID)

	if h.eventBroadcaster != nil {
		h.eventBroadcaster.Broadcast("navigation_completed", map[string]any{
			"task_id":  taskID,
			"robot_id": robotID,
			"store":    store.Name,
		})
	}

	arrivalText := fmt.Sprintf(arrivalTemplate, store.Name)
	if err := h.speak(ctx, robotID, arrivalText); err != nil {
		slog.Warn("mall_assistant speak arrival failed", "robot_id", robotID, "error", err)
	}

	if h.eventBroadcaster != nil {
		h.eventBroadcaster.Broadcast("visitor_interaction_finished", map[string]any{
			"task_id":  taskID,
			"robot_id": robotID,
		})
	}

	baseCoords := standbyCoordinates
	if baseNode != nil {
		baseCoords = baseNode.Coordinates.String()
	}
	h.createReturnTask(robotID, tenantID, operatorID, baseCoords)
	h.completeTask(taskID, robotID, tasks.StatusCompleted)
}

func (h *Handler) processRequest(ctx context.Context, robotID string, text string) (*mall.Store, error) {
	if h.cognitiveGateway == nil {
		return nil, nil
	}
	intentRes, err := h.cognitiveGateway.UnderstandIntent(ctx, cognitive.UnderstandIntentRequest{
		RobotID:  robotID,
		Text:     text,
		Language: defaultLanguage,
	})
	if err != nil {
		return nil, err
	}
	if intentRes.Intent != "find_store" {
		return nil, nil
	}
	storeName, _ := intentRes.Parameters["store_name"].(string)
	if storeName == "" {
		return nil, nil
	}
	return mall.FindStore(storeName)
}

func (h *Handler) speak(ctx context.Context, robotID string, text string) error {
	if h.cognitiveGateway == nil || h.bus == nil {
		return nil
	}
	res, err := h.cognitiveGateway.Synthesize(ctx, cognitive.SynthesizeRequest{
		RobotID:  robotID,
		Text:     text,
		Language: defaultLanguage,
	})
	if err != nil {
		return err
	}
	if res.AudioBase64 != "" {
		audioBytes, err := base64.StdEncoding.DecodeString(res.AudioBase64)
		if err == nil {
			_ = h.bus.PublishAudioOutput(robotID, audioBytes)
		}
	}
	return nil
}

func (h *Handler) createNavigateTask(robotID, tenantID, operatorID string, store *mall.Store, destNode *mall.NavNode, route []string, estimatedDistance float64) (string, error) {
	targetCoords := store.Coordinates
	if destNode != nil {
		targetCoords = destNode.Coordinates.String()
	}
	payload := map[string]any{
		"target_coordinates":  targetCoords,
		"store_name":         store.Name,
		"mall_id":            "default",
		"target_store":       store.Name,
		"destination_node_id": "",
		"route":              []string{},
		"estimated_distance": 0.0,
	}
	if destNode != nil {
		payload["destination_node_id"] = destNode.ID
	}
	if len(route) > 0 {
		payload["route"] = route
	}
	if estimatedDistance > 0 {
		payload["estimated_distance"] = estimatedDistance
	}
	payloadBytes, _ := json.Marshal(payload)
	t := &tasks.Task{
		ID:         uuid.New().String(),
		RobotID:    robotID,
		TenantID:   tenantID,
		Type:       "scenario",
		ScenarioID: "navigate_to_store",
		Payload:    payloadBytes,
		Status:     tasks.StatusPending,
		OperatorID: operatorID,
	}
	if t.TenantID == "" {
		t.TenantID = "default"
	}
	return t.ID, h.taskStore.Create(t)
}

func (h *Handler) waitForNavigation(ctx context.Context, navTaskID, robotID string) {
	deadline := time.Now().Add(navigateTaskWait)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t, _ := h.taskStore.Get(navTaskID)
			if t != nil && (t.Status == tasks.StatusCompleted || t.Status == tasks.StatusFailed || t.Status == tasks.StatusCancelled) {
				return
			}
		}
	}
}

func (h *Handler) createReturnTask(robotID, tenantID, operatorID string, baseCoordinates string) {
	if baseCoordinates == "" {
		baseCoordinates = standbyCoordinates
	}
	payload, _ := json.Marshal(map[string]any{
		"target_coordinates": baseCoordinates,
		"linear_x":           0.0,
		"linear_y":           0.0,
		"angular_z":          0.0,
		"duration_sec":       10,
	})
	t := &tasks.Task{
		ID:         uuid.New().String(),
		RobotID:    robotID,
		TenantID:   tenantID,
		Type:       "scenario",
		ScenarioID: "navigation",
		Payload:    payload,
		Status:     tasks.StatusPending,
		OperatorID: operatorID,
	}
	if t.TenantID == "" {
		t.TenantID = "default"
	}
	_ = h.taskStore.Create(t)
}

func (h *Handler) completeTask(taskID, robotID string, status tasks.Status) {
	now := time.Now()
	_ = h.taskStore.UpdateStatusAndCompletedAt(taskID, status, now)
	if h.onTaskCompleted != nil {
		h.onTaskCompleted(taskID, robotID, string(status))
	}
}

// SubmitVisitorRequest sends a visitor request to the active mall assistant for the robot.
func (h *Handler) SubmitVisitorRequest(robotID string, text string) bool {
	return h.requestRegistry.Submit(robotID, text)
}
