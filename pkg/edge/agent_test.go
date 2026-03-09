package edge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

func mustConnectBus(t *testing.T) *telemetry.Bus {
	bus, err := telemetry.NewBus("nats://localhost:4222")
	if err != nil {
		t.Skipf("NATS unavailable, skipping: %v", err)
	}
	t.Cleanup(func() { bus.Close() })
	return bus
}

func TestAgent_Sync_PostsHeartbeatAndPublishesCommands(t *testing.T) {
	bus := mustConnectBus(t)

	var reqBody HeartbeatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/edges/edge-001/heartbeat" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HeartbeatResponse{
			PendingCommands: []hal.Command{
				{RobotID: "edge-test-r1", Command: "safe_stop", Timestamp: time.Now()},
			},
		})
	}))
	defer server.Close()

	// Use unique robot ID to avoid NATS topic collision with pkg/control-plane tests (tenant_test, runner_test)
	// that also publish to commands.robots.r1 when running in parallel.
	robotID := "edge-test-r1"
	cfg := &Config{
		EdgeID:   "edge-001",
		CloudURL: server.URL,
		RobotIDs: []string{robotID},
	}
	agent := NewAgent(cfg, bus)

	var cmdReceived *hal.Command
	sub, err := bus.SubscribeCommands(robotID, func(cmd *hal.Command) {
		cmdReceived = cmd
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	err = agent.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if reqBody.EdgeID != "edge-001" {
		t.Errorf("heartbeat EdgeID: expected edge-001, got %s", reqBody.EdgeID)
	}
	time.Sleep(100 * time.Millisecond)
	if cmdReceived == nil || cmdReceived.Command != "safe_stop" {
		t.Errorf("expected safe_stop command published, got %v", cmdReceived)
	}
}

func TestAgent_Sync_Non200ReturnsError(t *testing.T) {
	bus := mustConnectBus(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	cfg := &Config{EdgeID: "e1", CloudURL: server.URL}
	agent := NewAgent(cfg, bus)

	err := agent.Sync(context.Background())
	if err == nil {
		t.Error("expected error on 500 response")
	}
}

func TestAgent_Sync_UnsafeCommandNotPublished(t *testing.T) {
	bus := mustConnectBus(t)

	// Use unique robot ID to avoid NATS topic collision with parallel tests.
	robotID := "edge-test-unsafe-r1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HeartbeatResponse{
			PendingCommands: []hal.Command{
				{RobotID: robotID, Command: "dangerous_unknown_command", Timestamp: time.Now()},
			},
		})
	}))
	defer server.Close()

	cfg := &Config{EdgeID: "e1", CloudURL: server.URL, RobotIDs: []string{robotID}}
	agent := NewAgent(cfg, bus)

	var received sync.Map
	sub, err := bus.SubscribeCommands(robotID, func(cmd *hal.Command) {
		received.Store("cmd", cmd)
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	err = agent.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if _, ok := received.Load("cmd"); ok {
		t.Error("unsafe command should not be published")
	}
}

func TestAgent_Sync_APIKeyInHeaderWhenConfigured(t *testing.T) {
	bus := mustConnectBus(t)

	var apiKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey = r.Header.Get("X-API-Key")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HeartbeatResponse{})
	}))
	defer server.Close()

	cfg := &Config{EdgeID: "e1", CloudURL: server.URL, APIKey: "secret-key-123"}
	agent := NewAgent(cfg, bus)

	err := agent.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if apiKey != "secret-key-123" {
		t.Errorf("expected X-API-Key header, got %q", apiKey)
	}
}
