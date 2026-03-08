package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sai-aurosy/platform/pkg/control-plane/arbiter"
	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

// Server is the Control Plane API server.
type Server struct {
	registry *registry.Store
	bus      *telemetry.Bus
}

// NewServer creates a new API server.
func NewServer(reg *registry.Store, bus *telemetry.Bus) *Server {
	return &Server{registry: reg, bus: bus}
}

// RegisterRoutes registers HTTP routes.
func (s *Server) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/robots", s.listRobots).Methods("GET")
	r.HandleFunc("/robots/{id}", s.getRobot).Methods("GET")
	r.HandleFunc("/robots/{id}/command", s.sendCommand).Methods("POST")
	r.HandleFunc("/telemetry/stream", s.telemetryStream).Methods("GET")
}

func (s *Server) listRobots(w http.ResponseWriter, r *http.Request) {
	robots := s.registry.List()
	json.NewEncoder(w).Encode(robots)
}

func (s *Server) getRobot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	robot := s.registry.Get(id)
	if robot == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(robot)
}

type commandRequest struct {
	Command    string `json:"command"`
	OperatorID string `json:"operator_id"`
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
	if !arbiter.SafetyAllow(cmd) {
		http.Error(w, "command not allowed", http.StatusForbidden)
		return
	}
	if err := s.bus.PublishCommand(cmd); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
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
