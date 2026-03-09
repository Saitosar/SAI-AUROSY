package sdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_ListRobots(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/robots" {
			t.Errorf("path = %s, want /v1/robots", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("X-API-Key = %s, want test-key", r.Header.Get("X-API-Key"))
		}
		json.NewEncoder(w).Encode([]Robot{
			{ID: "r1", Vendor: "agibot", Model: "X1"},
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-key")
	robots, err := client.ListRobots(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(robots) != 1 || robots[0].ID != "r1" {
		t.Errorf("robots = %+v", robots)
	}
}

func TestClient_GetRobot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/robots/r1" {
			t.Errorf("path = %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Robot{ID: "r1", Vendor: "agibot"})
	}))
	defer server.Close()

	client := New(server.URL, "test-key")
	robot, err := client.GetRobot(context.Background(), "r1")
	if err != nil {
		t.Fatal(err)
	}
	if robot.ID != "r1" || robot.Vendor != "agibot" {
		t.Errorf("robot = %+v", robot)
	}
}

func TestClient_CreateTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tasks" || r.Method != "POST" {
			t.Errorf("path = %s method = %s", r.URL.Path, r.Method)
		}
		var req CreateTaskRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.RobotID != "r1" || req.ScenarioID != "patrol" {
			t.Errorf("req = %+v", req)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Task{
			ID: "task-1", RobotID: "r1", ScenarioID: "patrol", Status: TaskStatusPending,
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-key")
	task, err := client.CreateTask(context.Background(), CreateTaskRequest{
		RobotID:    "r1",
		ScenarioID: "patrol",
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.ID != "task-1" || task.Status != TaskStatusPending {
		t.Errorf("task = %+v", task)
	}
}

func TestClient_ListWebhooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/webhooks" {
			t.Errorf("path = %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]Webhook{
			{ID: "wh1", URL: "https://example.com/hook", Events: []string{"task_completed"}},
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-key")
	webhooks, err := client.ListWebhooks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(webhooks) != 1 || webhooks[0].ID != "wh1" {
		t.Errorf("webhooks = %+v", webhooks)
	}
}

func TestClient_BaseURL(t *testing.T) {
	// baseURL with /v1 should not append
	client := New("http://localhost:8080/api/v1", "key")
	if client.baseURL != "http://localhost:8080/api/v1" {
		t.Errorf("baseURL = %s", client.baseURL)
	}

	// baseURL without /v1 should append /v1
	client2 := New("http://localhost:8080", "key")
	if client2.baseURL != "http://localhost:8080/v1" {
		t.Errorf("baseURL = %s", client2.baseURL)
	}
}
