package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/sai-aurosy/platform/pkg/control-plane/api"
	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	bus, err := telemetry.NewBus(natsURL)
	if err != nil {
		log.Fatalf("NATS: %v", err)
	}
	defer bus.Close()

	reg := registry.NewStore()
	seedRobots(reg)

	srv := api.NewServer(reg, bus)
	r := mux.NewRouter()
	srv.RegisterRoutes(r)

	addr := os.Getenv("CONTROL_PLANE_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("Control Plane listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}

func seedRobots(reg *registry.Store) {
	reg.Add(&hal.Robot{
		ID:              "x1-001",
		Vendor:          "agibot",
		Model:           "X1",
		AdapterEndpoint: "nats://localhost:4222",
		TenantID:        "default",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})
	reg.Add(&hal.Robot{
		ID:              "go2-001",
		Vendor:          "unitree",
		Model:           "Go2",
		AdapterEndpoint: "nats://localhost:4222",
		TenantID:        "default",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})
}
