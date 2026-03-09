package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sai-aurosy/platform/pkg/control-plane/arbiter"
	"github.com/sai-aurosy/platform/pkg/control-plane/audit"
	"github.com/sai-aurosy/platform/pkg/control-plane/auth"
	"github.com/sai-aurosy/platform/pkg/control-plane/coordinator"
	"github.com/sai-aurosy/platform/pkg/control-plane/analytics"
	"github.com/sai-aurosy/platform/pkg/control-plane/edges"
	"github.com/sai-aurosy/platform/pkg/control-plane/orchestration"
	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/control-plane/webhooks"
	"github.com/sai-aurosy/platform/pkg/control-plane/scenarios"
	"github.com/sai-aurosy/platform/pkg/control-plane/tasks"
	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

// Server is the Control Plane API server.
type Server struct {
	registry          registry.Store
	bus               *telemetry.Bus
	apiKeyStore       auth.APIKeyStore
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
}

// NewServer creates a new API server. apiKeyStore, coordinator, auditStore, webhookStore, webhookDispatcher, analyticsStore, and edgeStore are optional.
func NewServer(reg registry.Store, bus *telemetry.Bus, apiKeyStore auth.APIKeyStore, taskStore tasks.Store, scenarioCatalog *scenarios.Catalog, coord *coordinator.Coordinator, wfCatalog *orchestration.Catalog, wfRunStore orchestration.RunStore, wfRunner *orchestration.Runner, auditStore audit.Store, webhookStore webhooks.Store, webhookDispatcher *webhooks.Dispatcher, analyticsStore analytics.Store, edgeStore edges.Store) *Server {
	return &Server{
		registry:          reg,
		bus:               bus,
		apiKeyStore:       apiKeyStore,
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
	}
}

// RegisterRoutes registers HTTP routes.
func (s *Server) RegisterRoutes(r *mux.Router) {
	jwtMW := func(h http.Handler) http.Handler { return auth.MiddlewareWithAPIKeys(h, s.apiKeyStore) }
	opOrAdmin := auth.RequireRole(auth.RoleOperator, auth.RoleAdministrator)
	adminOnly := auth.RequireRole(auth.RoleAdministrator)

	// Versioned API v1
	v1 := r.PathPrefix("/v1").Subrouter()
	v1.Handle("/robots", jwtMW(opOrAdmin(http.HandlerFunc(s.listRobots)))).Methods("GET")
	v1.Handle("/robots", jwtMW(adminOnly(http.HandlerFunc(s.createRobot)))).Methods("POST")
	v1.Handle("/robots/{id}", jwtMW(opOrAdmin(http.HandlerFunc(s.getRobot)))).Methods("GET")
	v1.Handle("/robots/{id}", jwtMW(adminOnly(http.HandlerFunc(s.updateRobot)))).Methods("PUT")
	v1.Handle("/robots/{id}", jwtMW(adminOnly(http.HandlerFunc(s.deleteRobot)))).Methods("DELETE")
	v1.Handle("/robots/{id}/command", jwtMW(opOrAdmin(http.HandlerFunc(s.sendCommand)))).Methods("POST")
	v1.Handle("/telemetry/stream", jwtMW(opOrAdmin(http.HandlerFunc(s.telemetryStream)))).Methods("GET")
	v1.Handle("/zones", jwtMW(opOrAdmin(http.HandlerFunc(s.listZones)))).Methods("GET")
	v1.Handle("/zones/{id}", jwtMW(opOrAdmin(http.HandlerFunc(s.getZoneStatus)))).Methods("GET")
	v1.Handle("/workflows", jwtMW(opOrAdmin(http.HandlerFunc(s.listWorkflows)))).Methods("GET")
	v1.Handle("/workflows/{id}/run", jwtMW(opOrAdmin(http.HandlerFunc(s.runWorkflow)))).Methods("POST")
	v1.Handle("/workflow-runs", jwtMW(opOrAdmin(http.HandlerFunc(s.listWorkflowRuns)))).Methods("GET")
	v1.Handle("/workflow-runs/{id}", jwtMW(opOrAdmin(http.HandlerFunc(s.getWorkflowRun)))).Methods("GET")
	v1.Handle("/scenarios", jwtMW(opOrAdmin(http.HandlerFunc(s.listScenarios)))).Methods("GET")
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
	v1.Handle("/analytics/robots", jwtMW(opOrAdmin(http.HandlerFunc(s.listRobotAnalyticsSummaries)))).Methods("GET")
	v1.Handle("/analytics/robots/{id}/summary", jwtMW(opOrAdmin(http.HandlerFunc(s.getRobotAnalyticsSummary)))).Methods("GET")
	v1.Handle("/edges", jwtMW(opOrAdmin(http.HandlerFunc(s.listEdges)))).Methods("GET")
	v1.Handle("/edges/{id}", jwtMW(opOrAdmin(http.HandlerFunc(s.getEdge)))).Methods("GET")
	v1.Handle("/edges/{id}/heartbeat", http.HandlerFunc(s.edgeHeartbeat)).Methods("POST")

	// Legacy routes (deprecated)
	r.HandleFunc("/robots", deprecate(s.listRobots)).Methods("GET")
	r.HandleFunc("/robots/{id}", deprecate(s.getRobot)).Methods("GET")
	r.HandleFunc("/robots/{id}/command", deprecate(s.sendCommand)).Methods("POST")
	r.HandleFunc("/telemetry/stream", deprecate(s.telemetryStream)).Methods("GET")
}

func auditActor(operatorID string) string {
	if operatorID == "" {
		return "console"
	}
	return operatorID
}

func deprecate(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Deprecation", "true")
		w.Header().Set("X-API-Version", "v1")
		h(w, r)
	}
}

func (s *Server) listRobots(w http.ResponseWriter, r *http.Request) {
	robots := s.registry.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(robots)
}

func (s *Server) getRobot(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(robot)
}

type robotRequest struct {
	ID              string `json:"id"`
	Vendor          string `json:"vendor"`
	Model           string `json:"model"`
	AdapterEndpoint string   `json:"adapter_endpoint"`
	TenantID        string   `json:"tenant_id"`
	EdgeID         string   `json:"edge_id"`
	Capabilities   []string `json:"capabilities"`
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
		Capabilities:    caps,
	}
	s.registry.Add(robot)
	if s.auditStore != nil {
		_ = s.auditStore.Append(r.Context(), &audit.Entry{
			Actor:      "admin",
			Action:     "create",
			Resource:   "robot",
			ResourceID: robot.ID,
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
		robot.TenantID = req.TenantID
	}
	robot.EdgeID = req.EdgeID
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
	vars := mux.Vars(r)
	robotID := vars["id"]
	robot := s.registry.Get(robotID)
	if robot == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
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
			Timestamp:  time.Now(),
			Details:    string(details),
		})
	}
	if req.Command == "safe_stop" && s.webhookDispatcher != nil {
		s.webhookDispatcher.Dispatch(r.Context(), webhooks.EventSafeStop, map[string]any{
			"robot_id": robotID,
			"operator_id": auditActor(req.OperatorID),
		})
	}
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
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
	sub, err := s.bus.SubscribeAllTelemetry(func(t *hal.Telemetry) {
		data, _ := json.Marshal(t)
		w.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sub.Unsubscribe()
	<-r.Context().Done()
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
	result, err := s.workflowRunner.Run(workflowID, req.OperatorID)
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
	if s.workflowRunStore == nil {
		json.NewEncoder(w).Encode([]orchestration.WorkflowRun{})
		return
	}
	list, err := s.workflowRunStore.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

func (s *Server) listScenarios(w http.ResponseWriter, r *http.Request) {
	list := s.scenarioCatalog.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
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
	scenario, ok := s.scenarioCatalog.Get(req.ScenarioID)
	if !ok {
		http.Error(w, "scenario not found", http.StatusNotFound)
		return
	}
	robot := s.registry.Get(req.RobotID)
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
	t := &tasks.Task{
		ID:         uuid.New().String(),
		RobotID:    req.RobotID,
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
			Timestamp:  time.Now(),
			Details:    `{"robot_id":"` + t.RobotID + `","scenario_id":"` + t.ScenarioID + `"}`,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	filters := tasks.ListFilters{
		RobotID: r.URL.Query().Get("robot_id"),
		Status:  tasks.Status(r.URL.Query().Get("status")),
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
			Timestamp:  time.Now(),
			Details:    `{"robot_id":"` + t.RobotID + `"}`,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func (s *Server) listAudit(w http.ResponseWriter, r *http.Request) {
	if s.auditStore == nil {
		json.NewEncoder(w).Encode([]audit.Entry{})
		return
	}
	q := r.URL.Query()
	f := audit.ListFilters{
		RobotID: q.Get("robot_id"),
		Actor:   q.Get("actor"),
		Action:  q.Get("action"),
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
	if s.analyticsStore == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]*analytics.RobotSummary{})
		return
	}
	robots := s.registry.List()
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) getEdge(w http.ResponseWriter, r *http.Request) {
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
