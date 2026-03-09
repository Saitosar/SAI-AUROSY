package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
	"github.com/sai-aurosy/platform/pkg/control-plane/arbiter"
	"github.com/sai-aurosy/platform/pkg/control-plane/audit"
	"github.com/sai-aurosy/platform/pkg/control-plane/auth"
	"github.com/sai-aurosy/platform/pkg/control-plane/events"
	"github.com/sai-aurosy/platform/pkg/control-plane/commands"
	"github.com/sai-aurosy/platform/pkg/control-plane/cognitive"
	"github.com/sai-aurosy/platform/pkg/control-plane/conversations"
	"github.com/sai-aurosy/platform/pkg/control-plane/coordinator"
	"github.com/sai-aurosy/platform/pkg/control-plane/analytics"
	"github.com/sai-aurosy/platform/pkg/control-plane/edges"
	"github.com/sai-aurosy/platform/internal/mall"
	"github.com/sai-aurosy/platform/internal/simrobot"
	"github.com/sai-aurosy/platform/pkg/control-plane/mallassistant"
	"github.com/sai-aurosy/platform/pkg/control-plane/marketplace"
	"github.com/sai-aurosy/platform/pkg/control-plane/orchestration"
	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/control-plane/oauth"
	"github.com/sai-aurosy/platform/pkg/control-plane/scenarios"
	"github.com/sai-aurosy/platform/pkg/control-plane/webhooks"
	"github.com/sai-aurosy/platform/pkg/control-plane/tasks"
	"github.com/sai-aurosy/platform/pkg/control-plane/streaming"
	"github.com/sai-aurosy/platform/pkg/control-plane/tenants"
	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/telemetry"
	"github.com/sai-aurosy/platform/internal/robot"
)

// Server is the Control Plane API server.
type Server struct {
	registry          registry.Store
	bus               *telemetry.Bus
	apiKeyStore       auth.APIKeyStore
	apiKeyManager     auth.APIKeyManager
	taskStore         tasks.Store
	scenarioCatalog   *scenarios.Catalog
	coordinator       *coordinator.Coordinator
	workflowCatalog   *orchestration.Catalog
	workflowRunStore  orchestration.RunStore
	workflowRunner    *orchestration.Runner
	auditStore        audit.Store
	webhookStore      webhooks.Store
	webhookDispatcher *webhooks.Dispatcher
	analyticsStore    analytics.Store
	edgeStore         edges.Store
	tenantStore       tenants.Store
	oauthServer       *oauth.Server
	streamBuffer       *streaming.RingBuffer
	cognitiveGateway    cognitive.Gateway
	conversationCatalog *conversations.Catalog
	marketplaceStore    marketplace.Store
	idempotencyStore    commands.Store
	eventBroadcaster     *events.Broadcaster
	mallAssistantHandler *mallassistant.Handler
	mallService          *mall.Service
	robotStateProvider   robot.RobotStateProvider
	simRobotService      *simrobot.SimRobotService
}

// NewServer creates a new API server. apiKeyStore, coordinator, auditStore, webhookStore, webhookDispatcher, analyticsStore, edgeStore, tenantStore, oauthServer, idempotencyStore, eventBroadcaster, conversationCatalog, mallAssistantHandler, mallService, robotStateProvider, and simRobotService are optional.
func NewServer(reg registry.Store, bus *telemetry.Bus, apiKeyStore auth.APIKeyStore, taskStore tasks.Store, scenarioCatalog *scenarios.Catalog, coord *coordinator.Coordinator, wfCatalog *orchestration.Catalog, wfRunStore orchestration.RunStore, wfRunner *orchestration.Runner, auditStore audit.Store, webhookStore webhooks.Store, webhookDispatcher *webhooks.Dispatcher, analyticsStore analytics.Store, edgeStore edges.Store, tenantStore tenants.Store, oauthServer *oauth.Server, streamBuffer *streaming.RingBuffer, cognitiveGateway cognitive.Gateway, conversationCatalog *conversations.Catalog, marketplaceStore marketplace.Store, idempotencyStore commands.Store, eventBroadcaster *events.Broadcaster, mallAssistantHandler *mallassistant.Handler, mallService *mall.Service, robotStateProvider robot.RobotStateProvider, simRobotService *simrobot.SimRobotService) *Server {
	var apiKeyManager auth.APIKeyManager
	if apiKeyStore != nil {
		apiKeyManager, _ = apiKeyStore.(auth.APIKeyManager)
	}
	return &Server{
		registry:          reg,
		bus:               bus,
		apiKeyStore:       apiKeyStore,
		apiKeyManager:     apiKeyManager,
		taskStore:         taskStore,
		scenarioCatalog:   scenarioCatalog,
		coordinator:       coord,
		workflowCatalog:   wfCatalog,
		workflowRunStore:  wfRunStore,
		workflowRunner:    wfRunner,
		auditStore:        auditStore,
		webhookStore:      webhookStore,
		webhookDispatcher: webhookDispatcher,
		analyticsStore:    analyticsStore,
		edgeStore:         edgeStore,
		tenantStore:       tenantStore,
		oauthServer:       oauthServer,
		streamBuffer:       streamBuffer,
		cognitiveGateway:   cognitiveGateway,
		conversationCatalog: conversationCatalog,
		marketplaceStore:   marketplaceStore,
		idempotencyStore:     idempotencyStore,
		eventBroadcaster:     eventBroadcaster,
		mallAssistantHandler: mallAssistantHandler,
		mallService:          mallService,
		robotStateProvider:  robotStateProvider,
		simRobotService:     simRobotService,
	}
}

// RegisterRoutes registers HTTP routes.
func (s *Server) RegisterRoutes(r *mux.Router) {
	var oauthValidator auth.OAuthTokenValidator
	if s.oauthServer != nil {
		oauthValidator = s.oauthServer
	}
	jwtMW := func(h http.Handler) http.Handler {
		return auth.MiddlewareWithAPIKeysAndOAuth(h, s.apiKeyStore, oauthValidator)
	}
	opOrViewerOrAdmin := auth.RequireRole(auth.RoleOperator, auth.RoleAdministrator, auth.RoleViewer)
	opOrAdmin := auth.RequireRole(auth.RoleOperator, auth.RoleAdministrator)
	adminOnly := auth.RequireRole(auth.RoleAdministrator)
	opOrAdminOrSystem := auth.RequireRole(auth.RoleOperator, auth.RoleAdministrator, auth.RoleSystem)

	// Versioned API v1
	v1 := r.PathPrefix("/v1").Subrouter()
	v1.Handle("/me", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.getMe)))).Methods("GET")
	v1.Handle("/robots", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.listRobots)))).Methods("GET")
	v1.Handle("/robots", jwtMW(adminOnly(http.HandlerFunc(s.createRobot)))).Methods("POST")
	v1.Handle("/robots/{id}", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.getRobot)))).Methods("GET")
	v1.Handle("/robots/{id}/state", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.getRobotState)))).Methods("GET")
	v1.Handle("/robots/{id}", jwtMW(adminOnly(http.HandlerFunc(s.updateRobot)))).Methods("PUT")
	v1.Handle("/robots/{id}", jwtMW(adminOnly(http.HandlerFunc(s.deleteRobot)))).Methods("DELETE")
	v1.Handle("/robots/{id}/command", jwtMW(opOrAdmin(http.HandlerFunc(s.sendCommand)))).Methods("POST")
	v1.Handle("/telemetry/stream", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.telemetryStream)))).Methods("GET")
	v1.Handle("/events/stream", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.eventsStream)))).Methods("GET")
	v1.Handle("/zones", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.listZones)))).Methods("GET")
	v1.Handle("/zones/{id}", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.getZoneStatus)))).Methods("GET")
	v1.Handle("/workflows", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.listWorkflows)))).Methods("GET")
	v1.Handle("/workflows/{id}/run", jwtMW(opOrAdmin(http.HandlerFunc(s.runWorkflow)))).Methods("POST")
	v1.Handle("/workflow-runs", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.listWorkflowRuns)))).Methods("GET")
	v1.Handle("/workflow-runs/{id}", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.getWorkflowRun)))).Methods("GET")
	v1.Handle("/scenarios", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.listScenarios)))).Methods("GET")
	v1.Handle("/scenarios", jwtMW(adminOnly(http.HandlerFunc(s.createScenario)))).Methods("POST")
	v1.Handle("/scenarios/{id}", jwtMW(opOrAdmin(http.HandlerFunc(s.getScenario)))).Methods("GET")
	v1.Handle("/scenarios/{id}", jwtMW(adminOnly(http.HandlerFunc(s.updateScenario)))).Methods("PUT")
	v1.Handle("/scenarios/{id}", jwtMW(adminOnly(http.HandlerFunc(s.deleteScenario)))).Methods("DELETE")
	v1.Handle("/scenarios/mall_assistant/start", jwtMW(opOrAdmin(http.HandlerFunc(s.mallAssistantStart)))).Methods("POST")
	v1.Handle("/scenarios/mall_assistant/visitor-request", jwtMW(opOrAdmin(http.HandlerFunc(s.mallAssistantVisitorRequest)))).Methods("POST")
	v1.Handle("/tasks", jwtMW(opOrAdmin(http.HandlerFunc(s.listTasks)))).Methods("GET")
	v1.Handle("/tasks", jwtMW(opOrAdmin(http.HandlerFunc(s.createTask)))).Methods("POST")
	v1.Handle("/tasks/{id}", jwtMW(opOrAdmin(http.HandlerFunc(s.getTask)))).Methods("GET")
	v1.Handle("/tasks/{id}/cancel", jwtMW(opOrAdmin(http.HandlerFunc(s.cancelTask)))).Methods("POST")
	v1.Handle("/audit", jwtMW(opOrAdmin(http.HandlerFunc(s.listAudit)))).Methods("GET")
	v1.Handle("/webhooks", jwtMW(adminOnly(http.HandlerFunc(s.listWebhooks)))).Methods("GET")
	v1.Handle("/webhooks", jwtMW(adminOnly(http.HandlerFunc(s.createWebhook)))).Methods("POST")
	v1.Handle("/webhooks/{id}", jwtMW(adminOnly(http.HandlerFunc(s.getWebhook)))).Methods("GET")
	v1.Handle("/webhooks/{id}", jwtMW(adminOnly(http.HandlerFunc(s.updateWebhook)))).Methods("PUT")
	v1.Handle("/webhooks/{id}", jwtMW(adminOnly(http.HandlerFunc(s.deleteWebhook)))).Methods("DELETE")
	v1.Handle("/analytics/robots", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.listRobotAnalyticsSummaries)))).Methods("GET")
	v1.Handle("/analytics/robots/{id}/summary", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.getRobotAnalyticsSummary)))).Methods("GET")
	v1.Handle("/edges", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.listEdges)))).Methods("GET")
	v1.Handle("/edges/{id}", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.getEdge)))).Methods("GET")
	v1.Handle("/edges/{id}/heartbeat", jwtMW(opOrAdminOrSystem(http.HandlerFunc(s.edgeHeartbeat)))).Methods("POST")
	v1.Handle("/tenants", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.listTenants)))).Methods("GET")
	v1.Handle("/tenants", jwtMW(adminOnly(http.HandlerFunc(s.createTenant)))).Methods("POST")
	v1.Handle("/tenants/{id}", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.getTenant)))).Methods("GET")
	v1.Handle("/tenants/{id}", jwtMW(adminOnly(http.HandlerFunc(s.updateTenant)))).Methods("PUT")
	v1.Handle("/tenants/{id}", jwtMW(adminOnly(http.HandlerFunc(s.deleteTenant)))).Methods("DELETE")
	v1.Handle("/tenants/{id}/robots", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.listTenantRobots)))).Methods("GET")
	if s.apiKeyManager != nil {
		v1.Handle("/api-keys", jwtMW(opOrAdmin(http.HandlerFunc(s.createAPIKey)))).Methods("POST")
		v1.Handle("/api-keys", jwtMW(opOrAdmin(http.HandlerFunc(s.listAPIKeys)))).Methods("GET")
		v1.Handle("/api-keys/{id}", jwtMW(opOrAdmin(http.HandlerFunc(s.deleteAPIKey)))).Methods("DELETE")
	}
	if s.oauthServer != nil {
		v1.Handle("/oauth/clients", jwtMW(adminOnly(http.HandlerFunc(s.listOAuthClients)))).Methods("GET")
		v1.Handle("/oauth/clients", jwtMW(adminOnly(http.HandlerFunc(s.createOAuthClient)))).Methods("POST")
		v1.Handle("/oauth/clients/{client_id}", jwtMW(adminOnly(http.HandlerFunc(s.updateOAuthClient)))).Methods("PUT")
		v1.Handle("/oauth/clients/{client_id}", jwtMW(adminOnly(http.HandlerFunc(s.deleteOAuthClient)))).Methods("DELETE")
	}
	if s.marketplaceStore != nil {
		v1.Handle("/marketplace/categories", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.listMarketplaceCategories)))).Methods("GET")
		v1.Handle("/marketplace/scenarios", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.listMarketplaceScenarios)))).Methods("GET")
		v1.Handle("/marketplace/scenarios/{id}", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.getMarketplaceScenario)))).Methods("GET")
		v1.Handle("/marketplace/scenarios/{id}/rate", jwtMW(opOrAdmin(http.HandlerFunc(s.rateMarketplaceScenario)))).Methods("POST")
	}
	v1.Handle("/cognitive/navigate", jwtMW(opOrAdmin(http.HandlerFunc(s.cognitiveNavigate)))).Methods("POST")
	v1.Handle("/cognitive/recognize", jwtMW(opOrAdmin(http.HandlerFunc(s.cognitiveRecognize)))).Methods("POST")
	v1.Handle("/cognitive/plan", jwtMW(opOrAdmin(http.HandlerFunc(s.cognitivePlan)))).Methods("POST")
	v1.Handle("/cognitive/transcribe", jwtMW(opOrAdmin(http.HandlerFunc(s.cognitiveTranscribe)))).Methods("POST")
	v1.Handle("/cognitive/synthesize", jwtMW(opOrAdmin(http.HandlerFunc(s.cognitiveSynthesize)))).Methods("POST")
	v1.Handle("/cognitive/understand-intent", jwtMW(opOrAdmin(http.HandlerFunc(s.cognitiveUnderstandIntent)))).Methods("POST")
	if s.mallService != nil {
		v1.Handle("/malls/{mall_id}/map", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.getMallMap)))).Methods("GET")
		v1.Handle("/malls/{mall_id}/stores", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.listMallStores)))).Methods("GET")
		v1.Handle("/malls/{mall_id}/stores/{store_name}", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.getMallStore)))).Methods("GET")
		v1.Handle("/malls/{mall_id}/route", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.getMallRoute)))).Methods("GET")
	}
	if s.conversationCatalog != nil {
		v1.Handle("/conversations", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.listConversations)))).Methods("GET")
		v1.Handle("/conversations", jwtMW(adminOnly(http.HandlerFunc(s.createConversation)))).Methods("POST")
		v1.Handle("/conversations/{id}", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.getConversation)))).Methods("GET")
		v1.Handle("/conversations/{id}", jwtMW(adminOnly(http.HandlerFunc(s.updateConversation)))).Methods("PUT")
		v1.Handle("/conversations/{id}", jwtMW(adminOnly(http.HandlerFunc(s.deleteConversation)))).Methods("DELETE")
	}
	if s.simRobotService != nil {
		v1.Handle("/simrobots", jwtMW(opOrAdmin(http.HandlerFunc(s.simRobotCreate)))).Methods("POST")
		v1.Handle("/simrobots/{robot_id}/start", jwtMW(opOrAdmin(http.HandlerFunc(s.simRobotStart)))).Methods("POST")
		v1.Handle("/simrobots/{robot_id}/stop", jwtMW(opOrAdmin(http.HandlerFunc(s.simRobotStop)))).Methods("POST")
		v1.Handle("/simrobots/{robot_id}/reset", jwtMW(opOrAdmin(http.HandlerFunc(s.simRobotReset)))).Methods("POST")
		v1.Handle("/simrobots/{robot_id}/inject-failure", jwtMW(opOrAdmin(http.HandlerFunc(s.simRobotInjectFailure)))).Methods("POST")
		v1.Handle("/simrobots/{robot_id}/state", jwtMW(opOrViewerOrAdmin(http.HandlerFunc(s.simRobotGetState)))).Methods("GET")
	}
}

func auditActor(operatorID string) string {
	if operatorID == "" {
		return "console"
	}
	return operatorID
}

func auditActorFromClaims(claims *auth.Claims) string {
	if claims == nil {
		return "system"
	}
	if claims.Subject != "" {
		return claims.Subject
	}
	return "unknown"
}

func (s *Server) getMe(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "missing authorization", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"roles":     claims.GetRoles(),
		"tenant_id": claims.TenantID,
	})
}

// tenantFromRequest returns the effective tenant filter for the request.
// For operator: returns claims.TenantID (enforced); 403 if operator has no tenant_id.
// For administrator: returns query tenant_id or "" (all tenants).
// When claims is nil: 401 unless ALLOW_UNSAFE_NO_AUTH (then query param, for dev only).
func tenantFromRequest(r *http.Request) (tenantID string, statusCode int) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		if os.Getenv("ALLOW_UNSAFE_NO_AUTH") == "true" {
			return r.URL.Query().Get("tenant_id"), 0
		}
		return "", http.StatusUnauthorized
	}
	roles := claims.GetRoles()
	for _, role := range roles {
		if strings.EqualFold(role, auth.RoleAdministrator) {
			return r.URL.Query().Get("tenant_id"), 0
		}
	}
	// Operator: must have tenant_id in claims
	if claims.TenantID != "" {
		return claims.TenantID, 0
	}
	return "", http.StatusForbidden
}

func tenantOrError(w http.ResponseWriter, r *http.Request) (tenantID string, ok bool) {
	tenantID, status := tenantFromRequest(r)
	if status != 0 {
		msg := "missing authorization"
		if status == http.StatusForbidden {
			msg = "operator must have tenant_id in token"
		}
		http.Error(w, msg, status)
		return "", false
	}
	return tenantID, true
}

func (s *Server) listRobots(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	var robots []hal.Robot
	if tenantID != "" {
		robots = s.registry.ListByTenant(tenantID)
	} else {
		robots = s.registry.List()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(robots)
}

func (s *Server) getRobot(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	robot := s.registry.Get(id)
	if robot == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if tenantID != "" && robot.TenantID != tenantID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(robot)
}

func (s *Server) getRobotState(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	rb := s.registry.Get(id)
	if rb == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if tenantID != "" && rb.TenantID != tenantID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var state *robot.RobotStateResponse
	if s.robotStateProvider != nil {
		state = s.robotStateProvider.GetRobotState(id)
	} else {
		state = &robot.RobotStateResponse{
			RobotID:       id,
			State:         string(robot.StateIdle),
			StatusMessage: "idle",
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

func (s *Server) listTenants(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if s.tenantStore == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]tenants.Tenant{})
		return
	}
	list, err := s.tenantStore.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if tenantID != "" {
		var filtered []tenants.Tenant
		for _, t := range list {
			if t.ID == tenantID {
				filtered = append(filtered, t)
				break
			}
		}
		list = filtered
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) getTenant(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	if tenantID != "" && tenantID != id {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if s.tenantStore == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	t, err := s.tenantStore.Get(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if t == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

type createTenantRequest struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Config json.RawMessage `json:"config,omitempty"`
}

func (s *Server) createTenant(w http.ResponseWriter, r *http.Request) {
	if s.tenantStore == nil {
		http.Error(w, "tenants not configured", http.StatusServiceUnavailable)
		return
	}
	var req createTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.ID == "" || req.Name == "" {
		http.Error(w, "id and name required", http.StatusBadRequest)
		return
	}
	existing, _ := s.tenantStore.Get(req.ID)
	if existing != nil {
		http.Error(w, "tenant already exists", http.StatusConflict)
		return
	}
	t := &tenants.Tenant{ID: req.ID, Name: req.Name, Config: req.Config}
	if err := s.tenantStore.Create(t); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if s.auditStore != nil {
		claims := auth.GetClaims(r.Context())
		details, _ := json.Marshal(map[string]any{"name": t.Name})
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      auditActorFromClaims(claims),
			Action:     "create",
			Resource:   "tenant",
			ResourceID: t.ID,
			TenantID:   t.ID,
			Timestamp:  time.Now(),
			Details:    string(details),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

type updateTenantRequest struct {
	Name   string          `json:"name"`
	Config json.RawMessage `json:"config,omitempty"`
}

func (s *Server) updateTenant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	if s.tenantStore == nil {
		http.Error(w, "tenants not configured", http.StatusServiceUnavailable)
		return
	}
	existing, err := s.tenantStore.Get(id)
	if err != nil || existing == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var req updateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Config != nil {
		existing.Config = req.Config
	}
	if err := s.tenantStore.Update(existing); err != nil {
		if errors.Is(err, tenants.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if s.auditStore != nil {
		claims := auth.GetClaims(r.Context())
		details, _ := json.Marshal(map[string]any{"name": existing.Name})
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      auditActorFromClaims(claims),
			Action:     "update",
			Resource:   "tenant",
			ResourceID: id,
			TenantID:   id,
			Timestamp:  time.Now(),
			Details:    string(details),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

func (s *Server) deleteTenant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	if s.tenantStore == nil {
		http.Error(w, "tenants not configured", http.StatusServiceUnavailable)
		return
	}
	if id == "default" {
		http.Error(w, "cannot delete default tenant", http.StatusForbidden)
		return
	}
	robots := s.registry.ListByTenant(id)
	if len(robots) > 0 {
		http.Error(w, "cannot delete tenant with robots", http.StatusConflict)
		return
	}
	if err := s.tenantStore.Delete(id); err != nil {
		if errors.Is(err, tenants.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if s.auditStore != nil {
		claims := auth.GetClaims(r.Context())
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      auditActorFromClaims(claims),
			Action:     "delete",
			Resource:   "tenant",
			ResourceID: id,
			TenantID:   id,
			Timestamp:  time.Now(),
		})
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listTenantRobots(w http.ResponseWriter, r *http.Request) {
	effectiveTenant, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	vars := mux.Vars(r)
	tenantID := vars["id"]
	if tenantID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	if effectiveTenant != "" && effectiveTenant != tenantID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if s.tenantStore != nil {
		t, err := s.tenantStore.Get(tenantID)
		if err != nil || t == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	}
	robots := s.registry.ListByTenant(tenantID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(robots)
}

type createAPIKeyRequest struct {
	Name     string `json:"name"`
	Roles    string `json:"roles"`
	TenantID string `json:"tenant_id"`
}

func (s *Server) createAPIKey(w http.ResponseWriter, r *http.Request) {
	if s.apiKeyManager == nil {
		http.Error(w, "API key management requires database", http.StatusServiceUnavailable)
		return
	}
	var req createAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tenantID := req.TenantID
	if tenantID == "" {
		tenantID = "default"
	}
	roles := claims.GetRoles()
	isAdmin := false
	for _, role := range roles {
		if role == auth.RoleAdministrator {
			isAdmin = true
			break
		}
	}
	if !isAdmin && claims.TenantID != "" && tenantID != claims.TenantID {
		http.Error(w, "operator can only create keys for own tenant", http.StatusForbidden)
		return
	}
	resp, err := s.apiKeyManager.Create(r.Context(), &auth.CreateAPIKeyRequest{
		Name:     req.Name,
		Roles:    req.Roles,
		TenantID: tenantID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if s.auditStore != nil {
		details, _ := json.Marshal(map[string]any{"name": resp.Name, "roles": resp.Roles, "tenant_id": resp.TenantID})
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      auditActorFromClaims(claims),
			Action:     "create",
			Resource:   "api_key",
			ResourceID: resp.ID,
			TenantID:   resp.TenantID,
			Timestamp:  time.Now(),
			Details:    string(details),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	if s.apiKeyManager == nil {
		http.Error(w, "API key management requires database", http.StatusServiceUnavailable)
		return
	}
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	roles := claims.GetRoles()
	isAdmin := false
	for _, role := range roles {
		if role == auth.RoleAdministrator {
			isAdmin = true
			break
		}
	}
	tenantID := ""
	if !isAdmin && claims.TenantID != "" {
		tenantID = claims.TenantID
	}
	list, err := s.apiKeyManager.List(r.Context(), tenantID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) deleteAPIKey(w http.ResponseWriter, r *http.Request) {
	if s.apiKeyManager == nil {
		http.Error(w, "API key management requires database", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	roles := claims.GetRoles()
	isAdmin := false
	for _, role := range roles {
		if role == auth.RoleAdministrator {
			isAdmin = true
			break
		}
	}
	tenantID := ""
	if !isAdmin && claims.TenantID != "" {
		tenantID = claims.TenantID
	}
	if err := s.apiKeyManager.Delete(r.Context(), id, tenantID); err != nil {
		if errors.Is(err, auth.ErrAPIKeyNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if s.auditStore != nil {
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      auditActorFromClaims(claims),
			Action:     "delete",
			Resource:   "api_key",
			ResourceID: id,
			TenantID:   tenantID,
			Timestamp:  time.Now(),
		})
	}
	w.WriteHeader(http.StatusNoContent)
}

type createOAuthClientRequest struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
	TenantID     string   `json:"tenant_id"`
}

func (s *Server) listOAuthClients(w http.ResponseWriter, r *http.Request) {
	if s.oauthServer == nil {
		http.Error(w, "OAuth not configured", http.StatusServiceUnavailable)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	list, err := s.oauthServer.ListClients(r.Context(), tenantID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) createOAuthClient(w http.ResponseWriter, r *http.Request) {
	if s.oauthServer == nil {
		http.Error(w, "OAuth not configured", http.StatusServiceUnavailable)
		return
	}
	var req createOAuthClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.ClientID == "" || len(req.RedirectURIs) == 0 || len(req.Scopes) == 0 {
		http.Error(w, "client_id, redirect_uris, and scopes required", http.StatusBadRequest)
		return
	}
	if req.ClientSecret == "" {
		http.Error(w, "client_secret required", http.StatusBadRequest)
		return
	}
	tenantID := req.TenantID
	if tenantID == "" {
		tenantID = "default"
	}
	c := &oauth.Client{
		ClientID:       req.ClientID,
		ClientSecretHash: oauth.HashSHA256(req.ClientSecret),
		RedirectURIs:   req.RedirectURIs,
		Scopes:         req.Scopes,
		TenantID:       tenantID,
	}
	if err := s.oauthServer.CreateClient(r.Context(), c); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if s.auditStore != nil {
		claims := auth.GetClaims(r.Context())
		details, _ := json.Marshal(map[string]any{"client_id": c.ClientID, "tenant_id": c.TenantID})
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      auditActorFromClaims(claims),
			Action:     "create",
			Resource:   "oauth_client",
			ResourceID: c.ClientID,
			TenantID:   c.TenantID,
			Timestamp:  time.Now(),
			Details:    string(details),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

type updateOAuthClientRequest struct {
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
	TenantID     string   `json:"tenant_id"`
}

func (s *Server) updateOAuthClient(w http.ResponseWriter, r *http.Request) {
	if s.oauthServer == nil {
		http.Error(w, "OAuth not configured", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	clientID := vars["client_id"]
	if clientID == "" {
		http.Error(w, "client_id required", http.StatusBadRequest)
		return
	}
	existing, err := s.oauthServer.ListClients(r.Context(), "")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	var found *oauth.Client
	for _, c := range existing {
		if c.ClientID == clientID {
			found = c
			break
		}
	}
	if found == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var req updateOAuthClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if len(req.RedirectURIs) > 0 {
		found.RedirectURIs = req.RedirectURIs
	}
	if len(req.Scopes) > 0 {
		found.Scopes = req.Scopes
	}
	if req.TenantID != "" {
		found.TenantID = req.TenantID
	}
	if err := s.oauthServer.UpdateClient(r.Context(), found); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if s.auditStore != nil {
		claims := auth.GetClaims(r.Context())
		details, _ := json.Marshal(map[string]any{"redirect_uris": found.RedirectURIs, "scopes": found.Scopes})
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      auditActorFromClaims(claims),
			Action:     "update",
			Resource:   "oauth_client",
			ResourceID: clientID,
			TenantID:   found.TenantID,
			Timestamp:  time.Now(),
			Details:    string(details),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(found)
}

func (s *Server) deleteOAuthClient(w http.ResponseWriter, r *http.Request) {
	if s.oauthServer == nil {
		http.Error(w, "OAuth not configured", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	clientID := vars["client_id"]
	if clientID == "" {
		http.Error(w, "client_id required", http.StatusBadRequest)
		return
	}
	existing, err := s.oauthServer.ListClients(r.Context(), "")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	var tenantID string
	for _, c := range existing {
		if c.ClientID == clientID {
			tenantID = c.TenantID
			break
		}
	}
	if err := s.oauthServer.DeleteClient(r.Context(), clientID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if s.auditStore != nil {
		claims := auth.GetClaims(r.Context())
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      auditActorFromClaims(claims),
			Action:     "delete",
			Resource:   "oauth_client",
			ResourceID: clientID,
			TenantID:   tenantID,
			Timestamp:  time.Now(),
		})
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listMarketplaceCategories(w http.ResponseWriter, r *http.Request) {
	if s.marketplaceStore == nil {
		http.Error(w, "marketplace not configured", http.StatusServiceUnavailable)
		return
	}
	list, err := s.marketplaceStore.ListCategories(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) listMarketplaceScenarios(w http.ResponseWriter, r *http.Request) {
	if s.marketplaceStore == nil {
		http.Error(w, "marketplace not configured", http.StatusServiceUnavailable)
		return
	}
	opts := marketplace.ListOptions{
		Category: r.URL.Query().Get("category"),
		Search:   r.URL.Query().Get("search"),
		Sort:     r.URL.Query().Get("sort"),
	}
	if opts.Sort == "" {
		opts.Sort = "newest"
	}
	list, err := s.marketplaceStore.ListScenarios(r.Context(), opts)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) getMarketplaceScenario(w http.ResponseWriter, r *http.Request) {
	if s.marketplaceStore == nil {
		http.Error(w, "marketplace not configured", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	sc, err := s.marketplaceStore.GetScenario(r.Context(), id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if sc == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sc)
}

type rateScenarioRequest struct {
	Rating int `json:"rating"`
}

func (s *Server) rateMarketplaceScenario(w http.ResponseWriter, r *http.Request) {
	if s.marketplaceStore == nil {
		http.Error(w, "marketplace not configured", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = "default"
	}
	var req rateScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Rating < 1 || req.Rating > 5 {
		http.Error(w, "rating must be 1-5", http.StatusBadRequest)
		return
	}
	if err := s.marketplaceStore.RateScenario(r.Context(), id, tenantID, req.Rating); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type robotRequest struct {
	ID              string   `json:"id"`
	Vendor          string   `json:"vendor"`
	Model           string   `json:"model"`
	AdapterEndpoint string   `json:"adapter_endpoint"`
	TenantID        string   `json:"tenant_id"`
	EdgeID          string   `json:"edge_id"`
	Location        string   `json:"location"`
	Capabilities    []string `json:"capabilities"`
}

func (s *Server) createRobot(w http.ResponseWriter, r *http.Request) {
	var req robotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.ID == "" || req.Vendor == "" || req.Model == "" || req.AdapterEndpoint == "" {
		http.Error(w, "id, vendor, model, adapter_endpoint required", http.StatusBadRequest)
		return
	}
	if req.TenantID == "" {
		req.TenantID = "default"
	}
	if s.tenantStore != nil {
		t, err := s.tenantStore.Get(req.TenantID)
		if err != nil || t == nil {
			http.Error(w, "tenant not found", http.StatusBadRequest)
			return
		}
	}
	if s.registry.Get(req.ID) != nil {
		http.Error(w, "robot already exists", http.StatusConflict)
		return
	}
	caps := req.Capabilities
	if len(caps) == 0 {
		caps = []string{hal.CapWalk, hal.CapStand, hal.CapSafeStop, hal.CapReleaseControl, hal.CapCmdVel, hal.CapZeroMode, hal.CapPatrol, hal.CapNavigation}
	}
	robot := &hal.Robot{
		ID:              req.ID,
		Vendor:          req.Vendor,
		Model:           req.Model,
		AdapterEndpoint: req.AdapterEndpoint,
		TenantID:        req.TenantID,
		EdgeID:          req.EdgeID,
		Location:        req.Location,
		Capabilities:    caps,
	}
	s.registry.Add(robot)
	if s.auditStore != nil {
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      "admin",
			Action:     "create",
			Resource:   "robot",
			ResourceID: robot.ID,
			TenantID:   robot.TenantID,
			Timestamp:  time.Now(),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(robot)
}

func (s *Server) updateRobot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	robot := s.registry.Get(id)
	if robot == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var req robotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Vendor != "" {
		robot.Vendor = req.Vendor
	}
	if req.Model != "" {
		robot.Model = req.Model
	}
	if req.AdapterEndpoint != "" {
		robot.AdapterEndpoint = req.AdapterEndpoint
	}
	if req.TenantID != "" {
		if s.tenantStore != nil {
			t, err := s.tenantStore.Get(req.TenantID)
			if err != nil || t == nil {
				http.Error(w, "tenant not found", http.StatusBadRequest)
				return
			}
		}
		robot.TenantID = req.TenantID
	}
	robot.EdgeID = req.EdgeID
	robot.Location = req.Location
	if len(req.Capabilities) > 0 {
		robot.Capabilities = req.Capabilities
	}
	s.registry.Add(robot)
	if s.auditStore != nil {
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      "admin",
			Action:     "update",
			Resource:   "robot",
			ResourceID: id,
			TenantID:   robot.TenantID,
			Timestamp:  time.Now(),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(robot)
}

func (s *Server) deleteRobot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	robot := s.registry.Get(id)
	if robot == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !s.registry.Delete(id) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if s.auditStore != nil {
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      "admin",
			Action:     "delete",
			Resource:   "robot",
			ResourceID: id,
			TenantID:   robot.TenantID,
			Timestamp:  time.Now(),
		})
	}
	w.WriteHeader(http.StatusNoContent)
}

type commandRequest struct {
	Command    string         `json:"command"`
	Payload    map[string]any `json:"payload,omitempty"`
	OperatorID string         `json:"operator_id"`
}

func (s *Server) sendCommand(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	vars := mux.Vars(r)
	robotID := vars["id"]
	robot := s.registry.Get(robotID)
	if robot == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if tenantID != "" && robot.TenantID != tenantID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey != "" && s.idempotencyStore != nil {
		reserved, err := s.idempotencyStore.Reserve(r.Context(), idempotencyKey, robotID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !reserved {
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
			return
		}
	}
	var req commandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	cmd := &hal.Command{
		RobotID:    robotID,
		Command:    req.Command,
		Timestamp:  time.Now(),
		OperatorID: req.OperatorID,
	}
	if len(req.Payload) > 0 {
		payloadBytes, err := json.Marshal(req.Payload)
		if err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		cmd.Payload = payloadBytes
	}
	if !arbiter.SafetyAllow(cmd) {
		if s.auditStore != nil {
			details, _ := json.Marshal(map[string]any{"command": req.Command, "allowed": false})
			_ = s.auditStore.Append(r.Context(), &audit.Entry{
				Actor:      auditActor(req.OperatorID),
				Action:     "command",
				Resource:   "robot",
				ResourceID: robotID,
				TenantID:   robot.TenantID,
				Timestamp:  time.Now(),
				Details:    string(details),
			})
		}
		http.Error(w, "command not allowed", http.StatusForbidden)
		return
	}
	if robot.EdgeID != "" && s.edgeStore != nil {
		if err := s.edgeStore.EnqueueCommand(r.Context(), robot.EdgeID, robotID, cmd); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	} else if err := s.bus.PublishCommand(cmd); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if s.auditStore != nil {
		details, _ := json.Marshal(map[string]any{"command": req.Command, "allowed": true})
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      auditActor(req.OperatorID),
			Action:     "command",
			Resource:   "robot",
			ResourceID: robotID,
			TenantID:   robot.TenantID,
			Timestamp:  time.Now(),
			Details:    string(details),
		})
	}
	if req.Command == "safe_stop" {
		data := map[string]any{"robot_id": robotID, "operator_id": auditActor(req.OperatorID)}
		if s.webhookDispatcher != nil {
			s.webhookDispatcher.Dispatch(r.Context(), webhooks.EventSafeStop, data)
		}
		if s.eventBroadcaster != nil {
			s.eventBroadcaster.Broadcast(webhooks.EventSafeStop, data)
		}
	}
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func (s *Server) eventsStream(w http.ResponseWriter, r *http.Request) {
	if s.eventBroadcaster == nil {
		http.Error(w, "events stream not configured", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	_, ok = tenantOrError(w, r)
	if !ok {
		return
	}
	sub := s.eventBroadcaster.Subscribe()
	defer s.eventBroadcaster.Unsubscribe(sub)
	for {
		select {
		case e, ok := <-sub:
			if !ok {
				return
			}
			data, err := e.EventJSON()
			if err != nil {
				continue
			}
			w.Write([]byte("event: " + e.Type + "\ndata: " + string(data) + "\n\n"))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) telemetryStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	robotID := q.Get("robot_id")
	robotIDsParam := q.Get("robot_ids")
	var filterRobotIDs []string
	if robotID != "" {
		filterRobotIDs = []string{robotID}
	} else if robotIDsParam != "" {
		for _, id := range strings.Split(robotIDsParam, ",") {
			id = strings.TrimSpace(id)
			if id != "" {
				filterRobotIDs = append(filterRobotIDs, id)
			}
		}
	}
	var allowedRobotIDs map[string]bool
	if tenantID != "" {
		robots := s.registry.ListByTenant(tenantID)
		allowedRobotIDs = make(map[string]bool, len(robots))
		for _, robot := range robots {
			allowedRobotIDs[robot.ID] = true
		}
	}
	if len(filterRobotIDs) > 0 {
		for _, id := range filterRobotIDs {
			robot := s.registry.Get(id)
			if robot == nil {
				http.Error(w, "robot not found: "+id, http.StatusNotFound)
				return
			}
			if tenantID != "" && robot.TenantID != tenantID {
				http.Error(w, "robot not found: "+id, http.StatusNotFound)
				return
			}
		}
	}
	lastEventID := r.Header.Get("Last-Event-ID")
	if s.streamBuffer != nil && lastEventID != "" {
		replay := s.streamBuffer.GetSince(lastEventID)
		for _, ev := range replay {
			if len(filterRobotIDs) > 0 {
				found := false
				for _, fid := range filterRobotIDs {
					if ev.Telemetry.RobotID == fid {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			if tenantID != "" && !allowedRobotIDs[ev.Telemetry.RobotID] {
				continue
			}
			data, _ := json.Marshal(ev.Telemetry)
			w.Write([]byte("id: " + ev.ID + "\ndata: " + string(data) + "\n\n"))
			flusher.Flush()
		}
	}
	ch := make(chan *hal.Telemetry, streaming.DefaultBackpressureCapacity)
	handler := func(t *hal.Telemetry) {
		if len(filterRobotIDs) > 0 {
			found := false
			for _, fid := range filterRobotIDs {
				if t.RobotID == fid {
					found = true
					break
				}
			}
			if !found {
				return
			}
		}
		if tenantID != "" && !allowedRobotIDs[t.RobotID] {
			return
		}
		select {
		case ch <- t:
		default:
			<-ch
			streaming.IncDropped()
			ch <- t
		}
	}
	var sub *nats.Subscription
	var err error
	if len(filterRobotIDs) > 0 {
		sub, err = s.bus.SubscribeTelemetryMultiple(filterRobotIDs, handler)
	} else {
		sub, err = s.bus.SubscribeAllTelemetry(handler)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sub.Unsubscribe()
	for {
		select {
		case <-r.Context().Done():
			return
		case t := <-ch:
			id := t.Timestamp.UTC().Format(time.RFC3339Nano)
			data, _ := json.Marshal(t)
			w.Write([]byte("id: " + id + "\ndata: " + string(data) + "\n\n"))
			flusher.Flush()
		}
	}
}

func (s *Server) cognitiveNavigate(w http.ResponseWriter, r *http.Request) {
	if s.cognitiveGateway == nil {
		http.Error(w, "cognitive gateway not configured", http.StatusServiceUnavailable)
		return
	}
	var req cognitive.NavigateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.RobotID == "" {
		http.Error(w, "robot_id required", http.StatusBadRequest)
		return
	}
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if tenantID != "" {
		robot := s.registry.Get(req.RobotID)
		if robot == nil || robot.TenantID != tenantID {
			http.Error(w, "robot not found", http.StatusNotFound)
			return
		}
	}
	res, err := s.cognitiveGateway.Navigate(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (s *Server) cognitiveRecognize(w http.ResponseWriter, r *http.Request) {
	if s.cognitiveGateway == nil {
		http.Error(w, "cognitive gateway not configured", http.StatusServiceUnavailable)
		return
	}
	var req cognitive.RecognizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.RobotID == "" {
		http.Error(w, "robot_id required", http.StatusBadRequest)
		return
	}
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if tenantID != "" {
		robot := s.registry.Get(req.RobotID)
		if robot == nil || robot.TenantID != tenantID {
			http.Error(w, "robot not found", http.StatusNotFound)
			return
		}
	}
	res, err := s.cognitiveGateway.Recognize(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (s *Server) cognitivePlan(w http.ResponseWriter, r *http.Request) {
	if s.cognitiveGateway == nil {
		http.Error(w, "cognitive gateway not configured", http.StatusServiceUnavailable)
		return
	}
	var req cognitive.PlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.TaskType == "" {
		http.Error(w, "task_type required", http.StatusBadRequest)
		return
	}
	res, err := s.cognitiveGateway.Plan(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (s *Server) cognitiveTranscribe(w http.ResponseWriter, r *http.Request) {
	if s.cognitiveGateway == nil {
		http.Error(w, "cognitive gateway not configured", http.StatusServiceUnavailable)
		return
	}
	var req cognitive.TranscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.RobotID == "" {
		http.Error(w, "robot_id required", http.StatusBadRequest)
		return
	}
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if tenantID != "" {
		robot := s.registry.Get(req.RobotID)
		if robot == nil || robot.TenantID != tenantID {
			http.Error(w, "robot not found", http.StatusNotFound)
			return
		}
	}
	res, err := s.cognitiveGateway.Transcribe(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (s *Server) cognitiveSynthesize(w http.ResponseWriter, r *http.Request) {
	if s.cognitiveGateway == nil {
		http.Error(w, "cognitive gateway not configured", http.StatusServiceUnavailable)
		return
	}
	var req cognitive.SynthesizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.RobotID == "" {
		http.Error(w, "robot_id required", http.StatusBadRequest)
		return
	}
	if req.Text == "" {
		http.Error(w, "text required", http.StatusBadRequest)
		return
	}
	if req.Language == "" {
		http.Error(w, "language required", http.StatusBadRequest)
		return
	}
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if tenantID != "" {
		robot := s.registry.Get(req.RobotID)
		if robot == nil || robot.TenantID != tenantID {
			http.Error(w, "robot not found", http.StatusNotFound)
			return
		}
	}
	res, err := s.cognitiveGateway.Synthesize(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (s *Server) cognitiveUnderstandIntent(w http.ResponseWriter, r *http.Request) {
	if s.cognitiveGateway == nil {
		http.Error(w, "cognitive gateway not configured", http.StatusServiceUnavailable)
		return
	}
	var req cognitive.UnderstandIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.RobotID == "" {
		http.Error(w, "robot_id required", http.StatusBadRequest)
		return
	}
	if req.Text == "" {
		http.Error(w, "text required", http.StatusBadRequest)
		return
	}
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if tenantID != "" {
		robot := s.registry.Get(req.RobotID)
		if robot == nil || robot.TenantID != tenantID {
			http.Error(w, "robot not found", http.StatusNotFound)
			return
		}
	}
	res, err := s.cognitiveGateway.UnderstandIntent(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (s *Server) listConversations(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	list, err := s.conversationCatalog.ListForTenant(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) getConversation(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "conversation id required", http.StatusBadRequest)
		return
	}
	conv, err := s.conversationCatalog.GetForTenant(r.Context(), id, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if conv == nil {
		http.Error(w, "conversation not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conv)
}

func (s *Server) createConversation(w http.ResponseWriter, r *http.Request) {
	var conv conversations.Conversation
	if err := json.NewDecoder(r.Body).Decode(&conv); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if conv.ID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	if conv.Intent == "" {
		http.Error(w, "intent required", http.StatusBadRequest)
		return
	}
	if conv.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if conv.ResponseTemplate == "" && conv.ResponseProviderURL == "" {
		http.Error(w, "response_template or response_provider_url required", http.StatusBadRequest)
		return
	}
	if err := s.conversationCatalog.Create(r.Context(), &conv); err != nil {
		if errors.Is(err, conversations.ErrAlreadyExists) {
			http.Error(w, "conversation already exists", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(conv)
}

func (s *Server) updateConversation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "conversation id required", http.StatusBadRequest)
		return
	}
	var conv conversations.Conversation
	if err := json.NewDecoder(r.Body).Decode(&conv); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	conv.ID = id
	if conv.Intent == "" {
		http.Error(w, "intent required", http.StatusBadRequest)
		return
	}
	if conv.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if conv.ResponseTemplate == "" && conv.ResponseProviderURL == "" {
		http.Error(w, "response_template or response_provider_url required", http.StatusBadRequest)
		return
	}
	if err := s.conversationCatalog.Update(r.Context(), &conv); err != nil {
		if errors.Is(err, conversations.ErrNotFound) {
			http.Error(w, "conversation not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conv)
}

func (s *Server) deleteConversation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "conversation id required", http.StatusBadRequest)
		return
	}
	if err := s.conversationCatalog.Delete(r.Context(), id); err != nil {
		if errors.Is(err, conversations.ErrNotFound) {
			http.Error(w, "conversation not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listZones(w http.ResponseWriter, r *http.Request) {
	if s.coordinator == nil {
		json.NewEncoder(w).Encode([]coordinator.ZoneStatus{})
		return
	}
	list := s.coordinator.ListZoneStatuses()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) getZoneStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	zoneID := vars["id"]
	if zoneID == "" {
		http.Error(w, "zone id required", http.StatusBadRequest)
		return
	}
	if s.coordinator == nil {
		http.Error(w, "zones not configured", http.StatusServiceUnavailable)
		return
	}
	st := s.coordinator.GetZoneStatus(zoneID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(st)
}

func (s *Server) listWorkflows(w http.ResponseWriter, r *http.Request) {
	if s.workflowCatalog == nil {
		json.NewEncoder(w).Encode([]orchestration.Workflow{})
		return
	}
	list := s.workflowCatalog.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

type runWorkflowRequest struct {
	OperatorID string `json:"operator_id"`
}

func (s *Server) runWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]
	if workflowID == "" {
		http.Error(w, "workflow id required", http.StatusBadRequest)
		return
	}
	if s.workflowRunner == nil {
		http.Error(w, "workflows not configured", http.StatusServiceUnavailable)
		return
	}
	var req runWorkflowRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.OperatorID == "" {
		req.OperatorID = "console"
	}
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	result, err := s.workflowRunner.Run(r.Context(), workflowID, req.OperatorID, tenantID)
	if err != nil {
		if err == orchestration.ErrWorkflowNotFound {
			http.Error(w, "workflow not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.auditStore != nil && result != nil {
		runID := result.WorkflowRunID
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      auditActor(req.OperatorID),
			Action:     "run",
			Resource:   "workflow",
			ResourceID: runID,
			Timestamp:  time.Now(),
			Details:    `{"workflow_id":"` + workflowID + `"}`,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(result)
}

func (s *Server) listWorkflowRuns(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if s.workflowRunStore == nil {
		json.NewEncoder(w).Encode([]orchestration.WorkflowRun{})
		return
	}
	list, err := s.workflowRunStore.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if tenantID != "" {
		var filtered []orchestration.WorkflowRun
		for _, run := range list {
			if run.TenantID == tenantID {
				filtered = append(filtered, run)
			}
		}
		list = filtered
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) getWorkflowRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	if s.workflowRunStore == nil {
		http.Error(w, "workflow runs not configured", http.StatusServiceUnavailable)
		return
	}
	run, err := s.workflowRunStore.Get(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if run == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if tenantID != "" && run.TenantID != tenantID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

func (s *Server) listScenarios(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	list := s.scenarioCatalog.ListForTenant(tenantID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

type createScenarioRequest struct {
	ID                  string               `json:"id"`
	Name                string               `json:"name"`
	Description         string               `json:"description"`
	Steps               []scenarios.ScenarioStep `json:"steps"`
	RequiredCapabilities []string             `json:"required_capabilities"`
}

func (s *Server) createScenario(w http.ResponseWriter, r *http.Request) {
	var req createScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.ID == "" || req.Name == "" {
		http.Error(w, "id and name required", http.StatusBadRequest)
		return
	}
	if len(req.Steps) == 0 {
		http.Error(w, "at least one step required", http.StatusBadRequest)
		return
	}
	sc := &scenarios.Scenario{
		ID:                  req.ID,
		Name:                req.Name,
		Description:         req.Description,
		Steps:               req.Steps,
		RequiredCapabilities: req.RequiredCapabilities,
	}
	if err := s.scenarioCatalog.Create(r.Context(), sc); err != nil {
		if errors.Is(err, scenarios.ErrNotFound) {
			http.Error(w, "scenarios not persisted (no store)", http.StatusServiceUnavailable)
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, "scenario already exists", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sc)
}

func (s *Server) getScenario(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	sc, ok := s.scenarioCatalog.GetForTenant(id, tenantID)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sc)
}

type updateScenarioRequest struct {
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	Steps               []scenarios.ScenarioStep `json:"steps"`
	RequiredCapabilities []string               `json:"required_capabilities"`
}

func (s *Server) updateScenario(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	sc, ok := s.scenarioCatalog.Get(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var req updateScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Name != "" {
		sc.Name = req.Name
	}
	sc.Description = req.Description
	if len(req.Steps) > 0 {
		sc.Steps = req.Steps
	}
	if len(req.RequiredCapabilities) > 0 {
		sc.RequiredCapabilities = req.RequiredCapabilities
	}
	sc.ID = id
	if err := s.scenarioCatalog.Update(r.Context(), &sc); err != nil {
		if errors.Is(err, scenarios.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sc)
}

func (s *Server) deleteScenario(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	if err := s.scenarioCatalog.Delete(r.Context(), id); err != nil {
		if errors.Is(err, scenarios.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type mallAssistantStartRequest struct {
	RobotID    string `json:"robot_id"`
	OperatorID string `json:"operator_id"`
}

func (s *Server) mallAssistantStart(w http.ResponseWriter, r *http.Request) {
	if s.mallAssistantHandler == nil {
		http.Error(w, "mall assistant not configured", http.StatusServiceUnavailable)
		return
	}
	var req mallAssistantStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.RobotID == "" {
		http.Error(w, "robot_id required", http.StatusBadRequest)
		return
	}
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	robot := s.registry.Get(req.RobotID)
	if robot == nil {
		http.Error(w, "robot not found", http.StatusNotFound)
		return
	}
	if tenantID != "" && robot.TenantID != tenantID {
		http.Error(w, "robot not found", http.StatusNotFound)
		return
	}
	scenario, ok := s.scenarioCatalog.GetForTenant("mall_assistant", tenantID)
	if !ok {
		http.Error(w, "mall_assistant scenario not found", http.StatusNotFound)
		return
	}
	if !hal.HasCapability(robot, scenario.RequiredCapabilities) {
		http.Error(w, "robot lacks required capabilities for mall assistant", http.StatusBadRequest)
		return
	}
	hasRunning, err := s.taskStore.HasRunningForRobot(req.RobotID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if hasRunning {
		http.Error(w, "robot already has a running task", http.StatusConflict)
		return
	}
	operatorID := req.OperatorID
	if operatorID == "" {
		operatorID = "console"
	}
	taskTenantID := robot.TenantID
	if taskTenantID == "" {
		taskTenantID = "default"
	}
	t := &tasks.Task{
		ID:         uuid.New().String(),
		RobotID:    req.RobotID,
		TenantID:   taskTenantID,
		Type:       "scenario",
		ScenarioID: "mall_assistant",
		Status:     tasks.StatusPending,
		OperatorID: operatorID,
	}
	if err := s.taskStore.Create(t); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if s.auditStore != nil {
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      auditActor(operatorID),
			Action:     "create",
			Resource:   "task",
			ResourceID: t.ID,
			TenantID:   t.TenantID,
			Timestamp:  time.Now(),
			Details:    `{"robot_id":"` + t.RobotID + `","scenario_id":"mall_assistant"}`,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

type mallAssistantVisitorRequestPayload struct {
	RobotID string `json:"robot_id"`
	Text    string `json:"text"`
}

func (s *Server) mallAssistantVisitorRequest(w http.ResponseWriter, r *http.Request) {
	if s.mallAssistantHandler == nil {
		http.Error(w, "mall assistant not configured", http.StatusServiceUnavailable)
		return
	}
	var req mallAssistantVisitorRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.RobotID == "" {
		http.Error(w, "robot_id required", http.StatusBadRequest)
		return
	}
	if req.Text == "" {
		http.Error(w, "text required", http.StatusBadRequest)
		return
	}
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	robot := s.registry.Get(req.RobotID)
	if robot == nil {
		http.Error(w, "robot not found", http.StatusNotFound)
		return
	}
	if tenantID != "" && robot.TenantID != tenantID {
		http.Error(w, "robot not found", http.StatusNotFound)
		return
	}
	if !s.mallAssistantHandler.SubmitVisitorRequest(req.RobotID, req.Text) {
		http.Error(w, "no active mall assistant for this robot", http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

type createTaskRequest struct {
	RobotID    string          `json:"robot_id"`
	ScenarioID string          `json:"scenario_id"`
	Type       string          `json:"type"`
	Payload    json.RawMessage  `json:"payload,omitempty"`
	OperatorID string          `json:"operator_id"`
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.RobotID == "" {
		http.Error(w, "robot_id required", http.StatusBadRequest)
		return
	}
	if req.ScenarioID == "" {
		http.Error(w, "scenario_id required", http.StatusBadRequest)
		return
	}
	if s.registry.Get(req.RobotID) == nil {
		http.Error(w, "robot not found", http.StatusNotFound)
		return
	}
	robot := s.registry.Get(req.RobotID)
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if tenantID != "" && robot.TenantID != tenantID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	scenario, ok := s.scenarioCatalog.GetForTenant(req.ScenarioID, tenantID)
	if !ok {
		http.Error(w, "scenario not found", http.StatusNotFound)
		return
	}
	if !hal.HasCapability(robot, scenario.RequiredCapabilities) {
		http.Error(w, "robot lacks required capabilities for this scenario", http.StatusBadRequest)
		return
	}
	hasRunning, err := s.taskStore.HasRunningForRobot(req.RobotID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if hasRunning {
		http.Error(w, "robot already has a running task", http.StatusConflict)
		return
	}
	taskType := "scenario"
	if req.Type != "" {
		taskType = req.Type
	}
	if req.OperatorID == "" {
		req.OperatorID = "console"
	}
	taskTenantID := robot.TenantID
	if taskTenantID == "" {
		taskTenantID = "default"
	}
	t := &tasks.Task{
		ID:         uuid.New().String(),
		RobotID:    req.RobotID,
		TenantID:   taskTenantID,
		Type:       taskType,
		ScenarioID: req.ScenarioID,
		Payload:    req.Payload,
		Status:     tasks.StatusPending,
		OperatorID: req.OperatorID,
	}
	if err := s.taskStore.Create(t); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if s.auditStore != nil {
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      auditActor(req.OperatorID),
			Action:     "create",
			Resource:   "task",
			ResourceID: t.ID,
			TenantID:   t.TenantID,
			Timestamp:  time.Now(),
			Details:    `{"robot_id":"` + t.RobotID + `","scenario_id":"` + t.ScenarioID + `"}`,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	filters := tasks.ListFilters{
		RobotID:  r.URL.Query().Get("robot_id"),
		TenantID: tenantID,
		Status:   tasks.Status(r.URL.Query().Get("status")),
	}
	list, err := s.taskStore.List(filters)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	t, err := s.taskStore.Get(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if t == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if tenantID != "" && t.TenantID != tenantID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func (s *Server) cancelTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	t, err := s.taskStore.Get(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if t == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if tenantID != "" && t.TenantID != tenantID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if t.Status != tasks.StatusPending && t.Status != tasks.StatusRunning {
		http.Error(w, "task cannot be cancelled", http.StatusBadRequest)
		return
	}
	if err := s.taskStore.UpdateStatus(id, tasks.StatusCancelled); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	t.Status = tasks.StatusCancelled
	if s.auditStore != nil {
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      "console",
			Action:     "cancel",
			Resource:   "task",
			ResourceID: id,
			TenantID:   t.TenantID,
			Timestamp:  time.Now(),
			Details:    `{"robot_id":"` + t.RobotID + `"}`,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func (s *Server) listAudit(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if s.auditStore == nil {
		json.NewEncoder(w).Encode([]audit.Entry{})
		return
	}
	q := r.URL.Query()
	// For admin: tenantID may be "" (all); allow ?tenant_id= for filtering. Operator: tenantID from claims only.
	if tenantID == "" {
		tenantID = q.Get("tenant_id")
	}
	f := audit.ListFilters{
		RobotID:  q.Get("robot_id"),
		TenantID: tenantID,
		Actor:    q.Get("actor"),
		Action:   q.Get("action"),
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			f.Limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			f.Offset = n
		}
	}
	if v := q.Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.From = &t
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.To = &t
		}
	}
	list, err := s.auditStore.List(r.Context(), f)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) listWebhooks(w http.ResponseWriter, r *http.Request) {
	if s.webhookStore == nil {
		json.NewEncoder(w).Encode([]webhooks.Webhook{})
		return
	}
	list, err := s.webhookStore.List(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

type webhookRequest struct {
	URL     string   `json:"url"`
	Events  []string `json:"events"`
	Secret  string   `json:"secret"`
	Enabled *bool    `json:"enabled"`
}

func (s *Server) createWebhook(w http.ResponseWriter, r *http.Request) {
	if s.webhookStore == nil {
		http.Error(w, "webhooks not configured", http.StatusServiceUnavailable)
		return
	}
	var req webhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.URL == "" || len(req.Events) == 0 {
		http.Error(w, "url and events required", http.StatusBadRequest)
		return
	}
	wh := &webhooks.Webhook{
		URL:     req.URL,
		Events:  req.Events,
		Secret:  req.Secret,
		Enabled: true,
	}
	if req.Enabled != nil {
		wh.Enabled = *req.Enabled
	}
	if err := s.webhookStore.Create(r.Context(), wh); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(wh)
}

func (s *Server) getWebhook(w http.ResponseWriter, r *http.Request) {
	if s.webhookStore == nil {
		http.Error(w, "webhooks not configured", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]
	wb, err := s.webhookStore.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if wb == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wb)
}

func (s *Server) updateWebhook(w http.ResponseWriter, r *http.Request) {
	if s.webhookStore == nil {
		http.Error(w, "webhooks not configured", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]
	wb, err := s.webhookStore.Get(r.Context(), id)
	if err != nil || wb == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var req webhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.URL != "" {
		wb.URL = req.URL
	}
	if len(req.Events) > 0 {
		wb.Events = req.Events
	}
	if req.Secret != "" {
		wb.Secret = req.Secret
	}
	if req.Enabled != nil {
		wb.Enabled = *req.Enabled
	}
	if err := s.webhookStore.Update(r.Context(), wb); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wb)
}

func (s *Server) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	if s.webhookStore == nil {
		http.Error(w, "webhooks not configured", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]
	if err := s.webhookStore.Delete(r.Context(), id); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listRobotAnalyticsSummaries(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if s.analyticsStore == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]*analytics.RobotSummary{})
		return
	}
	var robots []hal.Robot
	if tenantID != "" {
		robots = s.registry.ListByTenant(tenantID)
	} else {
		robots = s.registry.List()
	}
	from, to := parseTimeRange(r, 24*time.Hour)
	var list []*analytics.RobotSummary
	for _, robot := range robots {
		sum, err := s.analyticsStore.RobotSummary(r.Context(), robot.ID, from, to)
		if err != nil {
			continue
		}
		list = append(list, sum)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) getRobotAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	if s.analyticsStore == nil {
		http.Error(w, "analytics not configured", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	robotID := vars["id"]
	if robotID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	robot := s.registry.Get(robotID)
	if robot == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if tenantID != "" && robot.TenantID != tenantID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	from, to := parseTimeRange(r, 24*time.Hour)
	sum, err := s.analyticsStore.RobotSummary(r.Context(), robotID, from, to)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sum)
}

func (s *Server) listEdges(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if s.edgeStore == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]edges.Edge{})
		return
	}
	list, err := s.edgeStore.ListEdges(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if tenantID != "" {
		robots := s.registry.ListByTenant(tenantID)
		edgeIDs := make(map[string]bool)
		for _, robot := range robots {
			if robot.EdgeID != "" {
				edgeIDs[robot.EdgeID] = true
			}
		}
		var filtered []edges.Edge
		for _, e := range list {
			if edgeIDs[e.ID] {
				filtered = append(filtered, e)
			}
		}
		list = filtered
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) getEdge(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	if s.edgeStore == nil {
		http.Error(w, "edges not configured", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	e, err := s.edgeStore.GetEdge(r.Context(), id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if e == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !ok {
		return
	}
	if tenantID != "" {
		robots := s.registry.ListByTenant(tenantID)
		allowed := false
		for _, robot := range robots {
			if robot.EdgeID == id {
				allowed = true
				break
			}
		}
		if !allowed {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(e)
}

type heartbeatRequest struct {
	EdgeID    string   `json:"edge_id"`
	Timestamp string   `json:"timestamp"`
	Robots    []string `json:"robots"`
}

func (s *Server) edgeHeartbeat(w http.ResponseWriter, r *http.Request) {
	if s.edgeStore == nil {
		http.Error(w, "edges not configured", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	edgeID := vars["id"]
	if edgeID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	var req heartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.EdgeID = edgeID
	now := time.Now()
	e := &edges.Edge{
		ID:            edgeID,
		LastHeartbeat: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.edgeStore.UpsertEdge(r.Context(), e); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	pending, err := s.edgeStore.FetchAndAckPendingCommands(r.Context(), edgeID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"pending_commands": pending})
}

func (s *Server) getMallMap(w http.ResponseWriter, r *http.Request) {
	if s.mallService == nil {
		http.Error(w, "mall service not configured", http.StatusServiceUnavailable)
		return
	}
	_, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	vars := mux.Vars(r)
	mallID := vars["mall_id"]
	if mallID == "" {
		http.Error(w, "mall_id required", http.StatusBadRequest)
		return
	}
	m, err := s.mallService.GetMallMap(r.Context(), mallID)
	if err != nil {
		if errors.Is(err, mall.ErrMallNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

func (s *Server) listMallStores(w http.ResponseWriter, r *http.Request) {
	if s.mallService == nil {
		http.Error(w, "mall service not configured", http.StatusServiceUnavailable)
		return
	}
	_, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	vars := mux.Vars(r)
	mallID := vars["mall_id"]
	if mallID == "" {
		http.Error(w, "mall_id required", http.StatusBadRequest)
		return
	}
	stores, err := s.mallService.ListStores(r.Context(), mallID)
	if err != nil {
		if errors.Is(err, mall.ErrMallNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stores)
}

func (s *Server) getMallStore(w http.ResponseWriter, r *http.Request) {
	if s.mallService == nil {
		http.Error(w, "mall service not configured", http.StatusServiceUnavailable)
		return
	}
	_, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	vars := mux.Vars(r)
	mallID := vars["mall_id"]
	storeName := vars["store_name"]
	if mallID == "" || storeName == "" {
		http.Error(w, "mall_id and store_name required", http.StatusBadRequest)
		return
	}
	node, err := s.mallService.FindStoreNode(r.Context(), mallID, storeName)
	if err != nil {
		if errors.Is(err, mall.ErrStoreNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, mall.ErrMallNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	m, _ := s.mallService.GetMallMap(r.Context(), mallID)
	var sl mall.StoreLocation
	for _, st := range m.Stores {
		if st.NodeID == node.ID {
			sl = st
			break
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"store_name": sl.StoreName,
		"floor_id":   sl.FloorID,
		"zone":       sl.Zone,
		"node":       node,
	})
}

func (s *Server) getMallRoute(w http.ResponseWriter, r *http.Request) {
	if s.mallService == nil {
		http.Error(w, "mall service not configured", http.StatusServiceUnavailable)
		return
	}
	_, ok := tenantOrError(w, r)
	if !ok {
		return
	}
	vars := mux.Vars(r)
	mallID := vars["mall_id"]
	fromID := r.URL.Query().Get("from")
	toID := r.URL.Query().Get("to")
	if mallID == "" || fromID == "" || toID == "" {
		http.Error(w, "mall_id, from, and to query params required", http.StatusBadRequest)
		return
	}
	route, dist, err := s.mallService.CalculateRoute(r.Context(), mallID, fromID, toID)
	if err != nil {
		if errors.Is(err, mall.ErrMallNotFound) || errors.Is(err, mall.ErrPathNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"route":             route,
		"estimated_distance": dist,
	})
}

func parseTimeRange(r *http.Request, defaultRange time.Duration) (from, to time.Time) {
	to = time.Now()
	from = to.Add(-defaultRange)
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t
		}
	}
	return from, to
}

// Sim robot handlers

func (s *Server) simRobotCreate(w http.ResponseWriter, r *http.Request) {
	var opts simrobot.CreateRobotOpts
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	robot, err := s.simRobotService.CreateRobot(opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"robot_id": robot.ID()})
}

func (s *Server) simRobotStart(w http.ResponseWriter, r *http.Request) {
	robotID := mux.Vars(r)["robot_id"]
	if err := s.simRobotService.Start(r.Context(), robotID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "started"})
}

func (s *Server) simRobotStop(w http.ResponseWriter, r *http.Request) {
	robotID := mux.Vars(r)["robot_id"]
	if err := s.simRobotService.Stop(robotID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "stopped"})
}

func (s *Server) simRobotReset(w http.ResponseWriter, r *http.Request) {
	robotID := mux.Vars(r)["robot_id"]
	if err := s.simRobotService.Reset(robotID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "reset"})
}

func (s *Server) simRobotInjectFailure(w http.ResponseWriter, r *http.Request) {
	robotID := mux.Vars(r)["robot_id"]
	var cfg simrobot.FailureConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := s.simRobotService.InjectFailure(robotID, &cfg); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "failure injected"})
}

func (s *Server) simRobotGetState(w http.ResponseWriter, r *http.Request) {
	robotID := mux.Vars(r)["robot_id"]
	state, err := s.simRobotService.GetState(robotID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}
