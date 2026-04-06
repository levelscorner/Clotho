package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/user/clotho/internal/api/dto"
	"github.com/user/clotho/internal/api/middleware"
	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/queue"
	"github.com/user/clotho/internal/store"
)

// ExecutionHandler handles execution endpoints.
type ExecutionHandler struct {
	executions store.ExecutionStore
	versions   store.PipelineVersionStore
	steps      store.StepResultStore
	queue      *queue.Queue
}

// NewExecutionHandler creates an ExecutionHandler.
func NewExecutionHandler(
	executions store.ExecutionStore,
	versions store.PipelineVersionStore,
	steps store.StepResultStore,
	q *queue.Queue,
) *ExecutionHandler {
	return &ExecutionHandler{
		executions: executions,
		versions:   versions,
		steps:      steps,
		queue:      q,
	}
}

// Routes registers execution routes on the given router.
func (h *ExecutionHandler) Routes(r chi.Router) {
	r.Post("/api/pipelines/{id}/execute", h.Execute)
	r.Get("/api/executions/{id}", h.Get)
	r.Get("/api/executions", h.List)
}

// Execute handles POST /api/pipelines/{id}/execute.
func (h *ExecutionHandler) Execute(w http.ResponseWriter, r *http.Request) {
	pipelineID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())

	// Get latest version for this pipeline
	pv, err := h.versions.GetLatest(r.Context(), pipelineID)
	if err != nil {
		writeError(w, http.StatusNotFound, "no versions found for pipeline")
		return
	}

	// Create execution record
	execution, err := h.executions.Create(r.Context(), domain.Execution{
		PipelineVersionID: pv.ID,
		TenantID:          tenantID,
		Status:            domain.StatusPending,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create execution")
		return
	}

	// Enqueue for background processing
	if err := h.queue.Submit(r.Context(), execution.ID, nil); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enqueue execution")
		return
	}

	writeJSON(w, http.StatusCreated, dto.ExecutionFromDomain(execution))
}

// Get handles GET /api/executions/{id}.
func (h *ExecutionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid execution ID")
		return
	}

	execution, err := h.executions.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "execution not found")
		return
	}

	steps, err := h.steps.ListByExecution(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list step results")
		return
	}

	writeJSON(w, http.StatusOK, dto.ExecutionWithSteps(execution, steps))
}

// List handles GET /api/executions.
func (h *ExecutionHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	limit := 20
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	executions, err := h.executions.ListByTenant(r.Context(), tenantID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list executions")
		return
	}

	writeJSON(w, http.StatusOK, dto.ExecutionsFromDomain(executions))
}
