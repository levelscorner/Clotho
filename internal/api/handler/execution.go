package handler

import (
	"encoding/json"
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
	pipelines  store.PipelineStore
	versions   store.PipelineVersionStore
	steps      store.StepResultStore
	queue      *queue.Queue
}

// NewExecutionHandler creates an ExecutionHandler.
func NewExecutionHandler(
	executions store.ExecutionStore,
	pipelines store.PipelineStore,
	versions store.PipelineVersionStore,
	steps store.StepResultStore,
	q *queue.Queue,
) *ExecutionHandler {
	return &ExecutionHandler{
		executions: executions,
		pipelines:  pipelines,
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

// executeRequest is the optional request body for POST /api/pipelines/{id}/execute.
type executeRequest struct {
	FromNodeID string `json:"from_node_id,omitempty"`
}

// Execute handles POST /api/pipelines/{id}/execute.
func (h *ExecutionHandler) Execute(w http.ResponseWriter, r *http.Request) {
	pipelineID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())

	// Tenant isolation — refuse executes on pipelines the caller does not own.
	// Responds 404 (not 403) to avoid ID enumeration.
	if _, err := h.pipelines.Get(r.Context(), pipelineID, tenantID); err != nil {
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}

	// Parse optional request body.
	var req executeRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	// Get latest version for this pipeline.
	pv, err := h.versions.GetLatest(r.Context(), pipelineID)
	if err != nil {
		writeError(w, http.StatusNotFound, "no versions found for pipeline")
		return
	}

	// Validate from_node_id (if provided) exists in the graph. An invalid ID
	// would otherwise surface as a confusing runtime error in the worker.
	if req.FromNodeID != "" {
		found := false
		for _, node := range pv.Graph.Nodes {
			if node.ID == req.FromNodeID {
				found = true
				break
			}
		}
		if !found {
			writeError(w, http.StatusBadRequest, "from_node_id is not a node in this pipeline")
			return
		}
	}

	// Create execution record.
	execution, err := h.executions.Create(r.Context(), domain.Execution{
		PipelineVersionID: pv.ID,
		TenantID:          tenantID,
		Status:            domain.StatusPending,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create execution")
		return
	}

	// Build payload with from_node_id if provided.
	var payload json.RawMessage
	if req.FromNodeID != "" {
		payload, err = json.Marshal(map[string]string{"from_node_id": req.FromNodeID})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to build execution payload")
			return
		}
	}

	// Enqueue for background processing.
	if err := h.queue.Submit(r.Context(), execution.ID, payload); err != nil {
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

	tenantID := middleware.TenantIDFromContext(r.Context())

	execution, err := h.executions.Get(r.Context(), id, tenantID)
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

// Cancel handles POST /api/executions/{id}/cancel.
func (h *ExecutionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid execution ID")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())

	if err := h.executions.Cancel(r.Context(), id, tenantID); err != nil {
		writeError(w, http.StatusNotFound, "execution not found or not cancellable")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
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
