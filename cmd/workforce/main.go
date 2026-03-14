// Package main runs the Workforce service: task engine, orchestration, analytics consumer,
// webhook delivery, and telemetry retention. It shares the same database and NATS as the
// Control Plane. Run with WORKFORCE_REMOTE=true on Control Plane to use this service.
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/sai-aurosy/platform/pkg/control-plane/analytics"
	"github.com/sai-aurosy/platform/pkg/control-plane/health"
	"github.com/sai-aurosy/platform/pkg/control-plane/analytics/retention"
	"github.com/sai-aurosy/platform/pkg/control-plane/coordinator"
	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/control-plane/scenarios"
	"github.com/sai-aurosy/platform/pkg/control-plane/tasks"
	"github.com/sai-aurosy/platform/pkg/control-plane/webhooks"
	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/secrets"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

func main() {
	ctx := context.Background()
	secretsProvider := secrets.Default(ctx)

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	bus, err := telemetry.NewBus(natsURL)
	if err != nil {
		log.Fatalf("NATS: %v", err)
	}
	defer bus.Close()

	driver := os.Getenv("REGISTRY_DB_DRIVER")
	if driver == "" {
		log.Fatal("REGISTRY_DB_DRIVER required for workforce (use sqlite, postgres, or libsql)")
	}
	dsn := secrets.GetSecretOrEnv(ctx, secretsProvider, "REGISTRY_DB_DSN")
	if dsn == "" {
		if driver == "sqlite" {
			dsn = "file::memory:?cache=shared"
		} else {
			log.Fatal("REGISTRY_DB_DSN required")
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
	reg := sqlStore
	db := sqlStore.DB()

	taskStore := tasks.NewSQLStore(db, driver)
	scenarioStore := scenarios.NewSQLStore(db, driver)
	scenarioCatalog := scenarios.NewCatalogWithStore(scenarioStore)
	coord := coordinator.NewCoordinator()

	webhookStore := webhooks.NewSQLStore(db, driver)
	webhookDeadLetter := webhooks.NewSQLDeadLetterStore(db, driver)
	webhookDispatcher := webhooks.NewDispatcherWithDeadLetter(webhookStore, webhookDeadLetter)
	analyticsStore := analytics.NewSQLStore(db, driver)

	taskRunner := tasks.NewRunnerWithCoordinator(taskStore, scenarioCatalog, reg, bus, coord, tasks.RunnerConfig{
		OnTaskCompleted: func(taskID, robotID, status string) {
			webhookDispatcher.Dispatch(context.Background(), webhooks.EventTaskCompleted, map[string]any{
				"task_id": taskID, "robot_id": robotID, "status": status,
			})
		},
		OnTaskStarted: func(taskID, robotID, scenarioID, zoneID string) {
			data := map[string]any{"task_id": taskID, "robot_id": robotID, "scenario_id": scenarioID}
			if zoneID != "" {
				data["zone_id"] = zoneID
			}
			webhookDispatcher.Dispatch(context.Background(), webhooks.EventTaskStarted, data)
		},
		OnZoneAcquired: func(robotID, zoneID, taskID string) {
			webhookDispatcher.Dispatch(context.Background(), webhooks.EventZoneAcquired, map[string]any{
				"robot_id": robotID, "zone_id": zoneID, "task_id": taskID,
			})
		},
		OnZoneReleased: func(robotID, zoneID, taskID string) {
			webhookDispatcher.Dispatch(context.Background(), webhooks.EventZoneReleased, map[string]any{
				"robot_id": robotID, "zone_id": zoneID, "task_id": taskID,
			})
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	healthAddr := os.Getenv("WORKFORCE_ADDR")
	if healthAddr == "" {
		healthAddr = ":9090"
	}
	healthHandler := health.NewHandler(bus, db)
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", healthHandler.Liveness)
	healthMux.HandleFunc("/ready", healthHandler.Readiness)
	healthSrv := &http.Server{Addr: healthAddr, Handler: healthMux}
	go func() {
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[health] %v", err)
		}
	}()

	go taskRunner.Run(ctx)
	go runRobotOnlineWatcher(ctx, bus, webhookDispatcher)
	go runTelemetryConsumer(ctx, bus, analyticsStore)
	go runTelemetryRetention(ctx, db, driver)

	graceSeconds := 25
	if n := os.Getenv("SHUTDOWN_GRACE_SECONDS"); n != "" {
		if s, err := strconv.Atoi(n); err == nil && s > 0 {
			graceSeconds = s
		}
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Printf("Shutting down (grace period=%ds)...", graceSeconds)
		time.Sleep(time.Duration(graceSeconds) * time.Second)
		cancel()
	}()

	log.Printf("Workforce running (NATS=%s, health=%s)", natsURL, healthAddr)
	<-ctx.Done()
}

func runRobotOnlineWatcher(ctx context.Context, bus *telemetry.Bus, dispatcher *webhooks.Dispatcher) {
	online := make(map[string]bool)
	sub, err := bus.SubscribeAllTelemetry(func(t *hal.Telemetry) {
		wasOnline := online[t.RobotID]
		online[t.RobotID] = t.Online
		if t.Online && !wasOnline {
			dispatcher.Dispatch(ctx, webhooks.EventRobotOnline, map[string]any{
				"robot_id": t.RobotID, "timestamp": t.Timestamp,
			})
		}
	})
	if err != nil {
		return
	}
	defer sub.Unsubscribe()
	<-ctx.Done()
}

func runTelemetryConsumer(ctx context.Context, bus *telemetry.Bus, store analytics.Store) {
	sub, err := bus.SubscribeAllTelemetry(func(t *hal.Telemetry) {
		if err := store.WriteTelemetry(ctx, t); err != nil {
			// Log but don't fail
		}
	})
	if err != nil {
		return
	}
	defer sub.Unsubscribe()
	<-ctx.Done()
}

func runTelemetryRetention(ctx context.Context, db *sql.DB, driver string) {
	cfg := retention.DefaultConfig()
	if n := os.Getenv("TELEMETRY_RETENTION_DAYS"); n != "" {
		if d, err := strconv.Atoi(n); err == nil && d > 0 {
			cfg.RetentionDays = d
		}
	}
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := retention.Run(ctx, db, driver, cfg); err != nil {
				log.Printf("[retention] %v", err)
			}
		}
	}
}
