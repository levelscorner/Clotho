package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/user/clotho/internal/api/handler"
	"github.com/user/clotho/internal/api/middleware"
	"github.com/user/clotho/internal/engine"
	"github.com/user/clotho/internal/llm"
	"github.com/user/clotho/internal/queue"
	"github.com/user/clotho/internal/storage"
	"github.com/user/clotho/internal/store"
)

// Deps holds all dependencies needed to construct the API router.
type Deps struct {
	Projects         store.ProjectStore
	Pipelines        store.PipelineStore
	PipelineVersions store.PipelineVersionStore
	Executions       store.ExecutionStore
	StepResults      store.StepResultStore
	Presets          store.PresetStore
	Credentials      store.CredentialStore
	Users            store.UserStore
	RefreshTokens    store.RefreshTokenStore
	LLMRegistry      *llm.ProviderRegistry
	Queue            *queue.Queue
	EventBus         *engine.EventBus
	FileStore         storage.Store
	JWTSecret         string
	JWTExpiry         time.Duration
	OllamaURL         string
	NoAuth            bool
	AcknowledgeNoAuth bool
}

// NewRouter creates a chi.Router with all middleware and routes mounted.
func NewRouter(deps Deps) chi.Router {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check (always public)
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Auth routes (public, no auth middleware)
	if deps.Users != nil && deps.RefreshTokens != nil && deps.JWTSecret != "" {
		handler.NewAuthHandler(deps.Users, deps.RefreshTokens, deps.JWTSecret, deps.JWTExpiry).Routes(r)
	}

	// Protected routes group
	r.Group(func(r chi.Router) {
		if deps.JWTSecret != "" {
			// JWT auth: inject user/tenant from token.
			// AuthWithConfig honors NoAuth + AcknowledgeNoAuth for dev bypass mode.
			r.Use(middleware.AuthWithConfig(middleware.AuthConfig{
				JWTSecret:         deps.JWTSecret,
				NoAuth:            deps.NoAuth,
				AcknowledgeNoAuth: deps.AcknowledgeNoAuth,
			}))
		} else {
			// Dev mode fallback: hardcoded tenant
			r.Use(middleware.Tenant)
		}

		handler.NewProjectHandler(deps.Projects).Routes(r)
		handler.NewPipelineHandler(deps.Pipelines, deps.PipelineVersions).Routes(r)
		execHandler := handler.NewExecutionHandler(deps.Executions, deps.PipelineVersions, deps.StepResults, deps.Queue)
		execHandler.Routes(r)
		r.Post("/api/executions/{id}/cancel", execHandler.Cancel)
		handler.NewPresetHandler(deps.Presets).Routes(r)
		handler.NewCredentialHandler(deps.Credentials).Routes(r)
		handler.NewProviderHandler(deps.LLMRegistry).Routes(r)
		handler.NewLLMHandler(deps.OllamaURL).Routes(r)
		handler.NewTemplateHandler().Routes(r)
		handler.NewStreamHandler(deps.EventBus).Routes(r)
		if deps.FileStore != nil {
			handler.NewFilesHandler(deps.FileStore).Routes(r)
		}
	})

	return r
}
