package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/api"
	"github.com/user/clotho/internal/auth"
	"github.com/user/clotho/internal/config"
	"github.com/user/clotho/internal/crypto"
	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/engine"
	"github.com/user/clotho/internal/llm"
	"github.com/user/clotho/internal/queue"
	"github.com/user/clotho/internal/store"
	"github.com/user/clotho/internal/store/postgres"
	"github.com/user/clotho/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Open Postgres pool and run migrations
	pool, err := postgres.New(ctx, cfg.DatabaseURL, migrations.FS)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Envelope encryption (optional)
	var envelope *crypto.Envelope
	if cfg.MasterKey != "" {
		envelope, err = crypto.NewEnvelope(cfg.MasterKey)
		if err != nil {
			slog.Error("failed to initialize envelope encryption", "error", err)
			os.Exit(1)
		}
		slog.Info("envelope encryption enabled")
	}

	// Create all stores
	projectStore := postgres.NewProjectStore(pool)
	pipelineStore := postgres.NewPipelineStore(pool)
	pipelineVersionStore := postgres.NewPipelineVersionStore(pool)
	executionStore := postgres.NewExecutionStore(pool)
	stepResultStore := postgres.NewStepResultStore(pool)
	presetStore := postgres.NewPresetStore(pool)
	credentialStore := postgres.NewCredentialStore(pool, envelope)
	userStore := postgres.NewUserStore(pool)
	refreshTokenStore := postgres.NewRefreshTokenStore(pool)
	workerID := uuid.New().String()
	jobStore := postgres.NewJobStore(pool, workerID)

	// Ensure admin user exists with configured password
	ensureAdminUser(ctx, userStore, cfg)

	// Create LLM provider registry
	llmRegistry := llm.NewRegistry()
	if cfg.OpenAIKey != "" {
		llmRegistry.Register("openai", llm.NewOpenAI(cfg.OpenAIKey))
	}
	if cfg.GeminiKey != "" {
		llmRegistry.Register("gemini", llm.NewGemini(cfg.GeminiKey))
	}
	if cfg.OpenRouterKey != "" {
		llmRegistry.Register("openrouter", llm.NewOpenRouter(cfg.OpenRouterKey))
	}
	llmRegistry.Register("ollama", llm.NewOllama(cfg.OllamaURL))

	slog.Info("LLM providers available", "providers", llmRegistry.List())

	// Create executor registry and register executors
	registry := engine.NewExecutorRegistry()
	registry.Register(domain.NodeTypeAgent, engine.NewAgentExecutor(llmRegistry, credentialStore))
	registry.Register(domain.NodeTypeTool, engine.NewToolExecutor())

	// Create event bus
	eventBus := engine.NewEventBus()

	// Create engine
	eng := engine.NewEngine(registry, eventBus, executionStore, stepResultStore)

	// Create queue
	q := queue.NewQueue(jobStore)

	// Create worker
	worker := queue.NewWorker(jobStore, executionStore, pipelineVersionStore, eng)

	deps := api.Deps{
		Projects:         projectStore,
		Pipelines:        pipelineStore,
		PipelineVersions: pipelineVersionStore,
		Executions:       executionStore,
		StepResults:      stepResultStore,
		Presets:          presetStore,
		Credentials:      credentialStore,
		Users:            userStore,
		RefreshTokens:    refreshTokenStore,
		LLMRegistry:      llmRegistry,
		Queue:            q,
		EventBus:         eventBus,
		JWTSecret:        cfg.JWTSecret,
		JWTExpiry:        cfg.JWTExpiry,
	}

	switch cfg.Mode {
	case "server":
		runServer(ctx, cfg, deps)
	case "worker":
		runWorker(ctx, worker)
	case "all":
		go runWorker(ctx, worker)
		runServer(ctx, cfg, deps)
	default:
		slog.Error("invalid mode", "mode", cfg.Mode)
		os.Exit(1)
	}
}

func runServer(ctx context.Context, cfg *config.Config, deps api.Deps) {
	router := api.NewRouter(deps)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		srv.Shutdown(shutdownCtx)
	}()

	slog.Info("starting server", "port", cfg.Port, "mode", cfg.Mode)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func runWorker(ctx context.Context, worker *queue.Worker) {
	slog.Info("starting worker")
	worker.Run(ctx)
	slog.Info("worker stopped")
}

func ensureAdminUser(ctx context.Context, users store.UserStore, cfg *config.Config) {
	const adminEmail = "admin@clotho.dev"

	hash, err := auth.HashPassword(cfg.AdminPassword)
	if err != nil {
		slog.Error("failed to hash admin password", "error", err)
		return
	}

	existing, err := users.GetByEmail(ctx, adminEmail)
	if err != nil {
		// User does not exist, create
		tenantID := uuid.New()
		if _, err := users.Create(ctx, domain.User{
			TenantID:     tenantID,
			Email:        adminEmail,
			Name:         "Admin",
			PasswordHash: hash,
			IsActive:     true,
		}); err != nil {
			slog.Error("failed to create admin user", "error", err)
			return
		}
		slog.Info("admin user created", "email", adminEmail)
		return
	}

	// User exists: update password if ADMIN_PASSWORD was explicitly set
	if os.Getenv("ADMIN_PASSWORD") != "" {
		if err := auth.ComparePassword(existing.PasswordHash, cfg.AdminPassword); err != nil {
			// Password differs, update it
			if updateErr := users.UpdatePassword(ctx, existing.ID, hash); updateErr != nil {
				slog.Error("failed to update admin password", "error", updateErr)
				return
			}
			slog.Info("admin user password updated via ADMIN_PASSWORD env", "email", adminEmail)
		}
	}
}
