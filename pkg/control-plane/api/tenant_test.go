package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/sai-aurosy/platform/pkg/control-plane/analytics"
	"github.com/sai-aurosy/platform/pkg/control-plane/auth"
	"github.com/sai-aurosy/platform/pkg/control-plane/edges"
	"github.com/sai-aurosy/platform/pkg/control-plane/orchestration"
	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/control-plane/scenarios"
	"github.com/sai-aurosy/platform/pkg/control-plane/tasks"
	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

// mockAPIKeyStore returns claims for test keys: t1, t2 (operators), admin (administrator).
type mockAPIKeyStore struct {
	keys map[string]*auth.Claims
}

func (m *mockAPIKeyStore) Validate(ctx context.Context, key string) (*auth.Claims, error) {
	return m.keys[key], nil
}

// mockAnalyticsStore returns empty summaries for any robot.
type mockAnalyticsStore struct{}

func (m *mockAnalyticsStore) WriteTelemetry(ctx context.Context, t *hal.Telemetry) error { return nil }
func (m *mockAnalyticsStore) RobotSummary(ctx context.Context, robotID string, from, to time.Time) (*analytics.RobotSummary, error) {
	return &analytics.RobotSummary{RobotID: robotID}, nil
}

func newMockAPIKeyStore() *mockAPIKeyStore {
	return &mockAPIKeyStore{
		keys: map[string]*auth.Claims{
			"key-t1": {
				Roles:    []string{"operator"},
				TenantID: "t1",
			},
			"key-t2": {
				Roles:    []string{"operator"},
				TenantID: "t2",
			},
			"key-admin": {
				Roles:    []string{"administrator"},
				TenantID: "",
			},
		},
	}
}

func setupTenantTestServer(t *testing.T) (*Server, *registry.MemoryStore) {
	reg := registry.NewMemoryStore()
	reg.Add(&hal.Robot{ID: "r1", Vendor: "test", Model: "X", AdapterEndpoint: "nats://x", TenantID: "t1", Capabilities: []string{hal.CapStand}})
	reg.Add(&hal.Robot{ID: "r2", Vendor: "test", Model: "X", AdapterEndpoint: "nats://x", TenantID: "t2", Capabilities: []string{hal.CapStand}})
	reg.Add(&hal.Robot{ID: "r3", Vendor: "test", Model: "X", AdapterEndpoint: "nats://x", TenantID: "t1", EdgeID: "edge-1", Capabilities: []string{hal.CapStand}})

	bus, _ := telemetry.NewBus("nats://localhost:4222")
	defer bus.Close()

	taskStore := tasks.NewMemoryStore()
	scenarioCatalog := scenarios.NewCatalog()
	wfCatalog := orchestration.NewCatalog()
	wfRunStore := orchestration.NewMemoryRunStore()
	wfRunner := orchestration.NewRunner(wfCatalog, wfRunStore, taskStore, scenarioCatalog, reg)

	apiKeyStore := newMockAPIKeyStore()
	analyticsStore := &mockAnalyticsStore{}
	edgeStore := edges.NewMemoryStore()
	srv := NewServer(reg, bus, apiKeyStore, taskStore, scenarioCatalog, nil, wfCatalog, wfRunStore, wfRunner, nil, nil, nil, analyticsStore, edgeStore, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	return srv, reg
}

func TestTenantEnforcement_SendCommand(t *testing.T) {
	srv, _ := setupTenantTestServer(t)
	claimsT1 := &auth.Claims{Roles: []string{"operator"}, TenantID: "t1"}

	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/v1/robots/{id}/command", func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), auth.ContextKeyClaims, claimsT1))
		srv.sendCommand(w, r)
	}).Methods("POST")

	// Operator t1 sending command to robot r2 (tenant t2) -> 404
	req := httptest.NewRequest("POST", "/v1/robots/r2/command", bytes.NewReader([]byte(`{"command":"stand_mode"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	muxRouter.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("operator t1 sending command to robot t2: expected 404, got %d", rec.Code)
	}

	// Operator t1 sending command to robot r1 (tenant t1) -> 202 (or 500 if NATS unavailable)
	req2 := httptest.NewRequest("POST", "/v1/robots/r1/command", bytes.NewReader([]byte(`{"command":"stand_mode"}`)))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	muxRouter.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusAccepted && rec2.Code != http.StatusInternalServerError {
		t.Errorf("operator t1 sending command to robot t1: expected 202 or 500, got %d", rec2.Code)
	}
}

func TestTenantEnforcement_CreateTask(t *testing.T) {
	srv, _ := setupTenantTestServer(t)

	// Operator t1 creates task for robot r2 (tenant t2) -> 403
	body, _ := json.Marshal(map[string]string{"robot_id": "r2", "scenario_id": "standby"})
	req := httptest.NewRequest("POST", "/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), auth.ContextKeyClaims, &auth.Claims{Roles: []string{"operator"}, TenantID: "t1"}))
	rec := httptest.NewRecorder()
	srv.createTask(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("operator t1 creating task for robot t2: expected 403, got %d", rec.Code)
	}

	// Operator t1 creates task for robot r1 (tenant t1) -> 201
	body2, _ := json.Marshal(map[string]string{"robot_id": "r1", "scenario_id": "standby"})
	req2 := httptest.NewRequest("POST", "/v1/tasks", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2 = req2.WithContext(context.WithValue(req2.Context(), auth.ContextKeyClaims, &auth.Claims{Roles: []string{"operator"}, TenantID: "t1"}))
	rec2 := httptest.NewRecorder()
	srv.createTask(rec2, req2)
	if rec2.Code != http.StatusCreated && rec2.Code != http.StatusInternalServerError {
		t.Errorf("operator t1 creating task for robot t1: expected 201, got %d", rec2.Code)
	}
}

func TestTenantEnforcement_CancelTask(t *testing.T) {
	_, reg := setupTenantTestServer(t)
	taskStore := tasks.NewMemoryStore()
	// Replace taskStore in server - we need to create a task in t2
	t2Task := &tasks.Task{
		ID:          "task-t2",
		RobotID:     "r2",
		TenantID:    "t2",
		Type:        "scenario",
		ScenarioID:  "standby",
		Status:      tasks.StatusPending,
	}
	taskStore.Create(t2Task)

	// Create server with our taskStore that has t2 task
	bus, _ := telemetry.NewBus("nats://localhost:4222")
	defer bus.Close()
	scenarioCatalog := scenarios.NewCatalog()
	wfCatalog := orchestration.NewCatalog()
	wfRunStore := orchestration.NewMemoryRunStore()
	wfRunner := orchestration.NewRunner(wfCatalog, wfRunStore, taskStore, scenarioCatalog, reg)
	srv2 := NewServer(reg, bus, newMockAPIKeyStore(), taskStore, scenarioCatalog, nil, wfCatalog, wfRunStore, wfRunner, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/v1/tasks/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), auth.ContextKeyClaims, &auth.Claims{Roles: []string{"operator"}, TenantID: "t1"}))
		srv2.cancelTask(w, r)
	}).Methods("POST")

	req := httptest.NewRequest("POST", "/v1/tasks/task-t2/cancel", nil)
	rec := httptest.NewRecorder()
	muxRouter.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("operator t1 cancelling task t2: expected 404, got %d", rec.Code)
	}
}

func TestTenantEnforcement_GetRobotAnalyticsSummary(t *testing.T) {
	srv, _ := setupTenantTestServer(t)

	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/v1/analytics/robots/{id}/summary", func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), auth.ContextKeyClaims, &auth.Claims{Roles: []string{"operator"}, TenantID: "t1"}))
		srv.getRobotAnalyticsSummary(w, r)
	}).Methods("GET")

	req := httptest.NewRequest("GET", "/v1/analytics/robots/r2/summary", nil)
	rec := httptest.NewRecorder()
	muxRouter.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("operator t1 getting analytics for robot t2: expected 404, got %d", rec.Code)
	}
}

func TestTenantEnforcement_ListRobotAnalyticsSummaries(t *testing.T) {
	srv, _ := setupTenantTestServer(t)

	req := httptest.NewRequest("GET", "/v1/analytics/robots", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ContextKeyClaims, &auth.Claims{Roles: []string{"operator"}, TenantID: "t1"}))
	rec := httptest.NewRecorder()
	srv.listRobotAnalyticsSummaries(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("list analytics: expected 200, got %d", rec.Code)
	}
	var list []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&list); err == nil {
		for _, item := range list {
			if rid, ok := item["robot_id"].(string); ok && rid == "r2" {
				t.Error("operator t1 should not see analytics for robot r2 (tenant t2)")
			}
		}
	}
}

func TestTenantEnforcement_ListWorkflowRuns(t *testing.T) {
	_, reg := setupTenantTestServer(t)
	wfRunStore := orchestration.NewMemoryRunStore()
	wfRunStore.Create(&orchestration.WorkflowRun{ID: "run1", WorkflowID: "w1", TenantID: "t1", Status: orchestration.WorkflowRunCompleted})
	wfRunStore.UpdateTenantID("run1", "t1")
	wfRunStore.Create(&orchestration.WorkflowRun{ID: "run2", WorkflowID: "w1", TenantID: "t2", Status: orchestration.WorkflowRunCompleted})
	wfRunStore.UpdateTenantID("run2", "t2")

	bus, _ := telemetry.NewBus("nats://localhost:4222")
	defer bus.Close()
	taskStore := tasks.NewMemoryStore()
	scenarioCatalog := scenarios.NewCatalog()
	wfCatalog := orchestration.NewCatalog()
	wfRunner := orchestration.NewRunner(wfCatalog, wfRunStore, taskStore, scenarioCatalog, reg)
	srv2 := NewServer(reg, bus, newMockAPIKeyStore(), taskStore, scenarioCatalog, nil, wfCatalog, wfRunStore, wfRunner, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest("GET", "/v1/workflow-runs", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ContextKeyClaims, &auth.Claims{Roles: []string{"operator"}, TenantID: "t1"}))
	rec := httptest.NewRecorder()
	srv2.listWorkflowRuns(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list workflow runs: expected 200, got %d", rec.Code)
	}
	var list []orchestration.WorkflowRun
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	for _, run := range list {
		if run.TenantID != "t1" {
			t.Errorf("operator t1 should only see runs for t1, got run with tenant %q", run.TenantID)
		}
	}
}

func TestTenantEnforcement_GetWorkflowRun(t *testing.T) {
	_, reg := setupTenantTestServer(t)
	wfRunStore := orchestration.NewMemoryRunStore()
	wfRunStore.Create(&orchestration.WorkflowRun{ID: "run-t2", WorkflowID: "w1", Status: orchestration.WorkflowRunCompleted})
	wfRunStore.UpdateTenantID("run-t2", "t2")

	bus, _ := telemetry.NewBus("nats://localhost:4222")
	defer bus.Close()
	taskStore := tasks.NewMemoryStore()
	scenarioCatalog := scenarios.NewCatalog()
	wfCatalog := orchestration.NewCatalog()
	wfRunner := orchestration.NewRunner(wfCatalog, wfRunStore, taskStore, scenarioCatalog, reg)
	srv2 := NewServer(reg, bus, newMockAPIKeyStore(), taskStore, scenarioCatalog, nil, wfCatalog, wfRunStore, wfRunner, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/v1/workflow-runs/{id}", func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), auth.ContextKeyClaims, &auth.Claims{Roles: []string{"operator"}, TenantID: "t1"}))
		srv2.getWorkflowRun(w, r)
	}).Methods("GET")

	req := httptest.NewRequest("GET", "/v1/workflow-runs/run-t2", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ContextKeyClaims, &auth.Claims{Roles: []string{"operator"}, TenantID: "t1"}))
	rec := httptest.NewRecorder()
	muxRouter.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("operator t1 getting workflow run t2: expected 404, got %d", rec.Code)
	}
}

func TestTenantEnforcement_ListEdges(t *testing.T) {
	_, reg := setupTenantTestServer(t)
	edgeStore := edges.NewMemoryStore()
	edgeStore.UpsertEdge(context.Background(), &edges.Edge{ID: "edge-1"})
	edgeStore.UpsertEdge(context.Background(), &edges.Edge{ID: "edge-2"})

	bus, _ := telemetry.NewBus("nats://localhost:4222")
	defer bus.Close()
	taskStore := tasks.NewMemoryStore()
	scenarioCatalog := scenarios.NewCatalog()
	wfCatalog := orchestration.NewCatalog()
	wfRunStore := orchestration.NewMemoryRunStore()
	wfRunner := orchestration.NewRunner(wfCatalog, wfRunStore, taskStore, scenarioCatalog, reg)
	srv2 := NewServer(reg, bus, newMockAPIKeyStore(), taskStore, scenarioCatalog, nil, wfCatalog, wfRunStore, wfRunner, nil, nil, nil, &mockAnalyticsStore{}, edgeStore, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest("GET", "/v1/edges", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ContextKeyClaims, &auth.Claims{Roles: []string{"operator"}, TenantID: "t1"}))
	rec := httptest.NewRecorder()
	srv2.listEdges(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list edges: expected 200, got %d", rec.Code)
	}
	var list []edges.Edge
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	for _, e := range list {
		if e.ID == "edge-2" {
			t.Error("operator t1 should not see edge-2 (only r3 has edge-1, r3 is t1)")
		}
	}
}
