package health

import (
	"database/sql"
	"net/http"

	"github.com/sai-aurosy/platform/pkg/telemetry"
)

// Handler provides health check endpoints.
type Handler struct {
	bus *telemetry.Bus
	db  *sql.DB
}

// NewHandler creates a health handler.
func NewHandler(bus *telemetry.Bus, db *sql.DB) *Handler {
	return &Handler{bus: bus, db: db}
}

// Liveness returns 200 if the process is alive.
func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// Readiness returns 200 if the service is ready to accept traffic.
func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	if h.bus != nil && !h.bus.IsConnected() {
		http.Error(w, "nats disconnected", http.StatusServiceUnavailable)
		return
	}
	if h.db != nil {
		if err := h.db.Ping(); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
