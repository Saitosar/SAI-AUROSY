package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sai-aurosy/platform/pkg/control-plane/api"
	"github.com/sai-aurosy/platform/pkg/control-plane/audit"
	"github.com/sai-aurosy/platform/pkg/control-plane/auth"
	"github.com/sai-aurosy/platform/pkg/control-plane/coordinator"
	"github.com/sai-aurosy/platform/pkg/control-plane/health"
	"github.com/sai-aurosy/platform/pkg/control-plane/orchestration"
	"github.com/sai-aurosy/platform/pkg/control-plane/httputil"
	"github.com/sai-aurosy/platform/pkg/control-plane/observability"
	"github.com/sai-aurosy/platform/pkg/control-plane/openapi"
	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/control-plane/scenarios"
	"github.com/sai-aurosy/platform/pkg/control-plane/analytics"
	"github.com/sai-aurosy/platform/pkg/control-plane/edges"
	"github.com/sai-aurosy/platform/pkg/control-plane/tasks"
	"github.com/sai-aurosy/platform/pkg/control-plane/webhooks"
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

	var reg registry.Store
	var apiKeyStore auth.APIKeyStore
	var db *sql.DB
	var driver string
	if driver = os.Getenv("REGISTRY_DB_DRIVER"); driver != "" {
		dsn := os.Getenv("REGISTRY_DB_DSN")
		if dsn == "" {
			if driver == "sqlite" {
				dsn = "file::memory:?cache=shared"
			} else {
				log.Fatal("REGISTRY_DB_DSN required when REGISTRY_DB_DRIVER is set")
			}
		}
		if driver == "postgres" {
			driver = "pgx"
		}
		sqlStore, err := registry.NewSQLStore(driver, dsn)
		if err != nil {
			log.Fatalf("registry: %v", err)
		}
		defer sqlStore.Close()
		reg = sqlStore
		db = sqlStore.DB()
		apiKeyStore = auth.NewSQLAPIKeyStore(db, driver)
	} else {
		reg = registry.NewMemoryStore()
	}
	seedRobots(reg)

	var taskStore tasks.Store
	if db != nil {
		taskStore = tasks.NewSQLStore(db, driver)
	} else {
		taskStore = tasks.NewMemoryStore()
	}
	scenarioCatalog := scenarios.NewCatalog()
	coord := coordinator.NewCoordinator()
	wfCatalog := orchestration.NewCatalog()
	var wfRunStore orchestration.RunStore
	if db != nil {
		wfRunStore = orchestration.NewSQLRunStore(db, driver)
	} else {
		wfRunStore = orchestration.NewMemoryRunStore()
	}
	wfRunner := orchestration.NewRunner(wfCatalog, wfRunStore, taskStore, scenarioCatalog, reg)

	var auditStore audit.Store
	var webhookStore webhooks.Store
	var analyticsStore analytics.Store
	var edgeStore edges.Store
	if db != nil {
		auditStore = audit.NewSQLStore(db, driver)
		webhookStore = webhooks.NewSQLStore(db, driver)
		analyticsStore = analytics.NewSQLStore(db, driver)
		edgeStore = edges.NewSQLStore(db, driver)
	} else {
		auditStore = audit.NewMemoryStore()
		webhookStore = webhooks.NewMemoryStore()
		edgeStore = edges.NewMemoryStore()
	}

	webhookDispatcher := webhooks.NewDispatcher(webhookStore)
	taskRunner := tasks.NewRunnerWithCoordinator(taskStore, scenarioCatalog, reg, bus, coord, tasks.RunnerConfig{
		OnTaskCompleted: func(taskID, robotID, status string) {
			webhookDispatcher.Dispatch(context.Background(), webhooks.EventTaskCompleted, map[string]any{
				"task_id":  taskID,
				"robot_id": robotID,
				"status":   status,
			})
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go taskRunner.Run(ctx)

	// Emit robot_online webhook when telemetry shows robot coming online
	go runRobotOnlineWatcher(ctx, bus, webhookDispatcher)

	// Store telemetry for analytics when DB is enabled
	if analyticsStore != nil {
		go runTelemetryConsumer(ctx, bus, analyticsStore)
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	srv := api.NewServer(reg, bus, apiKeyStore, taskStore, scenarioCatalog, coord, wfCatalog, wfRunStore, wfRunner, auditStore, webhookStore, webhookDispatcher, analyticsStore, edgeStore)
	r := mux.NewRouter()

	healthHandler := health.NewHandler(bus, db)
	r.HandleFunc("/health", healthHandler.Liveness).Methods("GET")
	r.HandleFunc("/ready", healthHandler.Readiness).Methods("GET")
	r.Handle("/metrics", observability.Handler()).Methods("GET")
	r.HandleFunc("/openapi.json", openapi.SpecHandler()).Methods("GET")
	r.HandleFunc("/swagger/", openapi.SwaggerUIHandler()).Methods("GET")

	srv.RegisterRoutes(r)

	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "*"
	}
	handler := cors.New(cors.Options{
		AllowedOrigins:   strings.Split(corsOrigins, ","),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-API-Key"},
		AllowCredentials: true,
	}).Handler(httputil.SecurityHeaders(httputil.RateLimit(
		observability.MetricsMiddleware(observability.LoggingMiddleware(r)),
	)))

	addr := os.Getenv("CONTROL_PLANE_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("Control Plane listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func runTelemetryConsumer(ctx context.Context, bus *telemetry.Bus, store analytics.Store) {
	sub, err := bus.SubscribeAllTelemetry(func(t *hal.Telemetry) {
		if err := store.WriteTelemetry(ctx, t); err != nil {
			// Log but don't fail - telemetry storage is best-effort
		}
	})
	if err != nil {
		return
	}
	defer sub.Unsubscribe()
	<-ctx.Done()
}

func runRobotOnlineWatcher(ctx context.Context, bus *telemetry.Bus, dispatcher *webhooks.Dispatcher) {
	if dispatcher == nil {
		return
	}
	online := make(map[string]bool)
	sub, err := bus.SubscribeAllTelemetry(func(t *hal.Telemetry) {
		wasOnline := online[t.RobotID]
		online[t.RobotID] = t.Online
		if t.Online && !wasOnline {
			dispatcher.Dispatch(ctx, webhooks.EventRobotOnline, map[string]any{
				"robot_id": t.RobotID,
				"timestamp": t.Timestamp,
			})
		}
	})
	if err != nil {
		return
	}
	defer sub.Unsubscribe()
	<-ctx.Done()
}

func seedRobots(reg registry.Store) {
	agibotCaps := []string{hal.CapWalk, hal.CapStand, hal.CapSafeStop, hal.CapReleaseControl, hal.CapCmdVel, hal.CapZeroMode, hal.CapPatrol, hal.CapNavigation}
	unitreeCaps := []string{hal.CapWalk, hal.CapStand, hal.CapSafeStop, hal.CapReleaseControl, hal.CapCmdVel, hal.CapZeroMode, hal.CapPatrol, hal.CapNavigation}
	reg.Add(&hal.Robot{
		ID:              "x1-001",
		Vendor:          "agibot",
		Model:           "X1",
		AdapterEndpoint: "nats://localhost:4222",
		TenantID:        "default",
		Capabilities:    agibotCaps,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})
	reg.Add(&hal.Robot{
		ID:              "go2-001",
		Vendor:          "unitree",
		Model:           "Go2",
		AdapterEndpoint: "nats://localhost:4222",
		TenantID:        "default",
		Capabilities:    unitreeCaps,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})
}
