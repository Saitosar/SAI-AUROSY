package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sai-aurosy/platform/pkg/control-plane/api"
	"github.com/sai-aurosy/platform/pkg/control-plane/audit"
	"github.com/sai-aurosy/platform/pkg/control-plane/auth"
	"github.com/sai-aurosy/platform/pkg/control-plane/events"
	"github.com/sai-aurosy/platform/pkg/control-plane/commands"
	"github.com/sai-aurosy/platform/pkg/control-plane/coordinator"
	"github.com/sai-aurosy/platform/pkg/control-plane/health"
	"github.com/sai-aurosy/platform/pkg/control-plane/orchestration"
	"github.com/sai-aurosy/platform/pkg/control-plane/httputil"
	"github.com/sai-aurosy/platform/pkg/control-plane/oauth"
	"github.com/sai-aurosy/platform/pkg/control-plane/observability"
	"github.com/sai-aurosy/platform/pkg/control-plane/openapi"
	"github.com/sai-aurosy/platform/pkg/control-plane/conversations"
	"github.com/sai-aurosy/platform/pkg/control-plane/cognitive"
	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/control-plane/scenarios"
	"github.com/sai-aurosy/platform/pkg/control-plane/streaming"
	"github.com/sai-aurosy/platform/pkg/control-plane/analytics"
	"github.com/sai-aurosy/platform/pkg/control-plane/analytics/retention"
	"github.com/sai-aurosy/platform/internal/mall"
	"github.com/sai-aurosy/platform/internal/robot"
	"github.com/sai-aurosy/platform/internal/simrobot"
	"github.com/sai-aurosy/platform/pkg/control-plane/edges"
	"github.com/sai-aurosy/platform/pkg/control-plane/mallassistant"
	"github.com/sai-aurosy/platform/pkg/control-plane/marketplace"
	"github.com/sai-aurosy/platform/pkg/control-plane/tasks"
	"github.com/sai-aurosy/platform/pkg/control-plane/tenants"
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

	var reg registry.Store
	var apiKeyStore auth.APIKeyStore
	var db *sql.DB
	var driver string
	if driver = os.Getenv("REGISTRY_DB_DRIVER"); driver != "" {
		dsn := secrets.GetSecretOrEnv(ctx, secretsProvider, "REGISTRY_DB_DSN")
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
	var scenarioCatalog *scenarios.Catalog
	if db != nil {
		scenarioStore := scenarios.NewSQLStore(db, driver)
		scenarioCatalog = scenarios.NewCatalogWithStore(scenarioStore)
	} else {
		scenarioCatalog = scenarios.NewCatalog()
	}
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

	var webhookDeadLetter webhooks.DeadLetterStore
	if db != nil {
		webhookDeadLetter = webhooks.NewSQLDeadLetterStore(db, driver)
	} else {
		webhookDeadLetter = webhooks.NewMemoryDeadLetterStore()
	}
	webhookDispatcher := webhooks.NewDispatcherWithDeadLetter(webhookStore, webhookDeadLetter)

	var tenantStore tenants.Store
	if db != nil {
		tenantStore = tenants.NewSQLStore(db, driver)
	} else {
		tenantStore = tenants.NewMemoryStore()
	}

	var idempotencyStore commands.Store
	if db != nil {
		idempotencyStore = commands.NewSQLStore(db, driver)
	} else {
		idempotencyStore = commands.NewMemoryStore()
	}

	var oauthServer *oauth.Server
	if db != nil {
		clientStore := oauth.NewSQLClientStore(db, driver)
		codeStore := oauth.NewSQLCodeStore(db, driver)
		tokenStore := oauth.NewSQLTokenStore(db, driver)
		baseURL := os.Getenv("OAUTH_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8080"
		}
		oauthServer = oauth.NewServer(clientStore, codeStore, tokenStore, baseURL)
	}

	// Enforce auth configuration: fail startup if auth required but not configured
	authConfigured := auth.AuthConfigured(apiKeyStore, oauthServer)
	authRequired := os.Getenv("AUTH_REQUIRED") != "false"
	allowUnsafeNoAuth := os.Getenv("ALLOW_UNSAFE_NO_AUTH") == "true"
	if !authConfigured {
		if authRequired && !allowUnsafeNoAuth {
			log.Fatal("Auth required but no JWT_SECRET, JWT_PUBLIC_KEY, or database (api_keys) configured. Set ALLOW_UNSAFE_NO_AUTH=true for development only.")
		}
		if allowUnsafeNoAuth {
			log.Print("WARNING: Running without authentication (ALLOW_UNSAFE_NO_AUTH=true). Do not use in production.")
		}
	}

	eventBroadcaster := events.NewBroadcaster()
	workforceRemote := os.Getenv("WORKFORCE_REMOTE") == "true"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cogCfg, err := cognitive.LoadConfig()
	if err != nil {
		log.Fatalf("cognitive config: %v", err)
	}
	cogGateway, err := cognitive.NewGateway(*cogCfg)
	if err != nil {
		log.Fatalf("cognitive gateway: %v", err)
	}

	mallRequestRegistry := mallassistant.NewVisitorRequestRegistry()
	mallRepo := mall.NewMemoryRepository("scenarios/data/mall_map.json")
	mallService := mall.NewService(mallRepo)
	var mallAssistantHandler *mallassistant.Handler
	if !workforceRemote {
		mallAssistantHandler = mallassistant.NewHandler(bus, cogGateway, taskStore, eventBroadcaster, mallRequestRegistry, mallassistant.HandlerConfig{
			MallService: mallService,
			OnTaskCompleted: func(taskID, robotID, status string) {
				data := map[string]any{"task_id": taskID, "robot_id": robotID, "status": status}
				webhookDispatcher.Dispatch(context.Background(), webhooks.EventTaskCompleted, data)
				eventBroadcaster.Broadcast(webhooks.EventTaskCompleted, data)
			},
		})
	}

	var executionEngine *robot.RobotExecutionEngine
	if !workforceRemote {
		stateManager := robot.NewStateManager()
		navExecutor := robot.NewNavigationExecutor(bus, stateManager, 1.0, 60*time.Second)
		taskExecutor := robot.NewTaskExecutor(navExecutor)
		executionEngine = robot.NewExecutionEngine(stateManager, taskExecutor, bus, robot.ExecutionEngineConfig{
			EventBroadcaster:    eventBroadcaster,
			TaskStore:          taskStore,
			TimeoutAsCompletion: true,
			OnTaskCompleted: func(taskID, robotID, status string) {
				data := map[string]any{"task_id": taskID, "robot_id": robotID, "status": status}
				webhookDispatcher.Dispatch(context.Background(), webhooks.EventTaskCompleted, data)
				eventBroadcaster.Broadcast(webhooks.EventTaskCompleted, data)
			},
		})
	}

	var taskRunner *tasks.Runner
	if !workforceRemote {
		taskRunner = tasks.NewRunnerWithCoordinator(taskStore, scenarioCatalog, reg, bus, coord, tasks.RunnerConfig{
		MallAssistantRunner: mallAssistantHandler,
		ExecutionEngine:     executionEngine,
		OnTaskCompleted: func(taskID, robotID, status string) {
			data := map[string]any{"task_id": taskID, "robot_id": robotID, "status": status}
			webhookDispatcher.Dispatch(context.Background(), webhooks.EventTaskCompleted, data)
			eventBroadcaster.Broadcast(webhooks.EventTaskCompleted, data)
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
				"robot_id": robotID,
				"zone_id":  zoneID,
				"task_id":  taskID,
			})
		},
		OnZoneReleased: func(robotID, zoneID, taskID string) {
			webhookDispatcher.Dispatch(context.Background(), webhooks.EventZoneReleased, map[string]any{
				"robot_id": robotID,
				"zone_id":  zoneID,
				"task_id":  taskID,
			})
		},
		})
		go taskRunner.Run(ctx)
	}

	// Simulated robot harness: create and start sim-001 when enabled
	simRobotEnabled := os.Getenv("SIMROBOT_ENABLED") != "false"
	var simRobotService *simrobot.SimRobotService
	if simRobotEnabled && !workforceRemote {
		simRobotService = simrobot.NewSimRobotService(bus, reg)
		if _, err := simRobotService.CreateRobot(simrobot.CreateRobotOpts{
			RobotID:   "sim-001",
			TenantID:  "default",
			RobotType: "simulated",
		}); err != nil {
			log.Printf("WARNING: simrobot create failed: %v", err)
		} else if err := simRobotService.Start(ctx, "sim-001"); err != nil {
			log.Printf("WARNING: simrobot start failed: %v", err)
		} else {
			log.Print("Simulated robot sim-001 started")
		}
	}

	if !workforceRemote {
		// Emit robot_online webhook when telemetry shows robot coming online
		go runRobotOnlineWatcher(ctx, bus, webhookDispatcher, eventBroadcaster)

		// Telemetry retention: aggregate and delete old samples
		if db != nil && analyticsStore != nil {
			go runTelemetryRetention(ctx, db, driver)
		}

		// Store telemetry for analytics when DB is enabled
		if analyticsStore != nil {
			go runTelemetryConsumer(ctx, bus, analyticsStore)
		}
	}

	// Clean up expired idempotency keys (24h TTL)
	go runIdempotencyCleanup(ctx, idempotencyStore)

	// Stream buffer for SSE reconnect (Last-Event-ID)
	streamBuf := streaming.NewRingBuffer(streaming.DefaultBufferCapacity)
	go runStreamBufferPopulator(ctx, bus, streamBuf)

	shutdownTimeout := 30 * time.Second
	if n := os.Getenv("SHUTDOWN_TIMEOUT"); n != "" {
		if sec, err := strconv.Atoi(n); err == nil && sec > 0 {
			shutdownTimeout = time.Duration(sec) * time.Second
		}
	}

	var httpServer *http.Server
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Printf("Shutting down (timeout=%v)...", shutdownTimeout)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()
		if httpServer != nil {
			if err := httpServer.Shutdown(shutdownCtx); err != nil {
				log.Printf("HTTP shutdown: %v", err)
			}
		}
		cancel()
	}()

	var marketplaceStore marketplace.Store
	if db != nil {
		marketplaceStore = marketplace.NewSQLStore(db, driver)
	}
	var conversationCatalog *conversations.Catalog
	if db != nil {
		convStore := conversations.NewSQLStore(db, driver)
		conversationCatalog = conversations.NewCatalog(convStore)
	} else {
		convStore := conversations.NewMemoryStore()
		conversationCatalog = conversations.NewCatalog(convStore)
	}
	var robotStateProvider robot.RobotStateProvider
	if executionEngine != nil {
		robotStateProvider = executionEngine
	}
	srv := api.NewServer(reg, bus, apiKeyStore, taskStore, scenarioCatalog, coord, wfCatalog, wfRunStore, wfRunner, auditStore, webhookStore, webhookDispatcher, analyticsStore, edgeStore, tenantStore, oauthServer, streamBuf, cogGateway, conversationCatalog, marketplaceStore, idempotencyStore, eventBroadcaster, mallAssistantHandler, mallService, robotStateProvider, simRobotService)
	r := mux.NewRouter()

	healthHandler := health.NewHandler(bus, db)
	r.HandleFunc("/health", healthHandler.Liveness).Methods("GET")
	r.HandleFunc("/ready", healthHandler.Readiness).Methods("GET")
	r.Handle("/metrics", observability.Handler()).Methods("GET")
	r.HandleFunc("/openapi.json", openapi.SpecHandler()).Methods("GET")
	r.HandleFunc("/api/openapi.json", openapi.SpecHandler()).Methods("GET")
	r.HandleFunc("/swagger/", openapi.SwaggerUIHandler()).Methods("GET")
	r.HandleFunc("/api/docs", openapi.SwaggerUIHandler()).Methods("GET")

	if oauthServer != nil {
		r.HandleFunc("/oauth/authorize", oauthServer.HandleAuthorize).Methods("GET")
		r.HandleFunc("/oauth/token", oauthServer.HandleToken).Methods("POST")
		r.HandleFunc("/oauth/revoke", oauthServer.HandleRevoke).Methods("POST")
	}

	srv.RegisterRoutes(r)

	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "*"
	}
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	shutdownTracer, err := observability.InitTracer("", otlpEndpoint)
	if err != nil {
		log.Printf("WARNING: OpenTelemetry tracer init failed (tracing disabled): %v", err)
	} else {
		defer shutdownTracer()
	}

	handler := cors.New(cors.Options{
		AllowedOrigins:   strings.Split(corsOrigins, ","),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-API-Key", "Idempotency-Key"},
		AllowCredentials: true,
	}).Handler(httputil.SecurityHeaders(httputil.RateLimit(
		observability.MetricsMiddleware(observability.TracingMiddleware(observability.LoggingMiddleware(r))),
	)))

	addr := os.Getenv("CONTROL_PLANE_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	httpServer = &http.Server{Addr: addr, Handler: handler}
	log.Printf("Control Plane listening on %s", addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func runStreamBufferPopulator(ctx context.Context, bus *telemetry.Bus, buf *streaming.RingBuffer) {
	sub, err := bus.SubscribeAllTelemetry(func(t *hal.Telemetry) {
		buf.Add(t)
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
			// Log but don't fail - telemetry storage is best-effort
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

func runIdempotencyCleanup(ctx context.Context, store commands.Store) {
	if store == nil {
		return
	}
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := store.Cleanup(ctx, 24*time.Hour); err != nil {
				log.Printf("[idempotency] cleanup: %v", err)
			}
		}
	}
}

func runRobotOnlineWatcher(ctx context.Context, bus *telemetry.Bus, dispatcher *webhooks.Dispatcher, broadcaster *events.Broadcaster) {
	if dispatcher == nil && broadcaster == nil {
		return
	}
	online := make(map[string]bool)
	sub, err := bus.SubscribeAllTelemetry(func(t *hal.Telemetry) {
		wasOnline := online[t.RobotID]
		online[t.RobotID] = t.Online
		if t.Online && !wasOnline {
			data := map[string]any{"robot_id": t.RobotID, "timestamp": t.Timestamp}
			if dispatcher != nil {
				dispatcher.Dispatch(ctx, webhooks.EventRobotOnline, data)
			}
			if broadcaster != nil {
				broadcaster.Broadcast(webhooks.EventRobotOnline, data)
			}
		}
	})
	if err != nil {
		return
	}
	defer sub.Unsubscribe()
	<-ctx.Done()
}

func seedRobots(reg registry.Store) {
	agibotCaps := []string{hal.CapWalk, hal.CapStand, hal.CapSafeStop, hal.CapReleaseControl, hal.CapCmdVel, hal.CapZeroMode, hal.CapPatrol, hal.CapNavigation, hal.CapSpeech}
	unitreeCaps := []string{hal.CapWalk, hal.CapStand, hal.CapSafeStop, hal.CapReleaseControl, hal.CapCmdVel, hal.CapZeroMode, hal.CapPatrol, hal.CapNavigation, hal.CapSpeech}
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
