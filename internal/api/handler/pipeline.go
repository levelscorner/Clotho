package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/user/clotho/internal/api/dto"
	"github.com/user/clotho/internal/api/middleware"
	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/store"
)

// PipelineHandler handles pipeline and pipeline version endpoints.
type PipelineHandler struct {
	pipelines store.PipelineStore
	projects  store.ProjectStore
	versions  store.PipelineVersionStore
}

// NewPipelineHandler creates a PipelineHandler.
func NewPipelineHandler(pipelines store.PipelineStore, projects store.ProjectStore, versions store.PipelineVersionStore) *PipelineHandler {
	return &PipelineHandler{pipelines: pipelines, projects: projects, versions: versions}
}

// Routes registers pipeline routes on the given router.
func (h *PipelineHandler) Routes(r chi.Router) {
	r.Get("/api/projects/{projectID}/pipelines", h.ListByProject)
	r.Post("/api/projects/{projectID}/pipelines", h.Create)
	r.Get("/api/pipelines/{id}", h.Get)
	r.Put("/api/pipelines/{id}", h.Update)
	r.Delete("/api/pipelines/{id}", h.Delete)
	r.Get("/api/pipelines/{id}/export", h.Export)
	r.Post("/api/pipelines/{id}/import", h.Import)
	r.Post("/api/pipelines/{id}/versions", h.SaveVersion)
	r.Get("/api/pipelines/{id}/versions", h.ListVersions)
	r.Get("/api/pipelines/{id}/versions/latest", h.GetLatestVersion)
	r.Get("/api/pipelines/{id}/versions/{version}", h.GetVersion)
}

// assertPipelineOwned confirms the pipeline exists and belongs to the tenant.
// Responds with 404 on miss and returns false; callers bail out immediately.
func (h *PipelineHandler) assertPipelineOwned(w http.ResponseWriter, r *http.Request, pipelineID uuid.UUID) bool {
	tenantID := middleware.TenantIDFromContext(r.Context())
	if _, err := h.pipelines.Get(r.Context(), pipelineID, tenantID); err != nil {
		writeError(w, http.StatusNotFound, "pipeline not found")
		return false
	}
	return true
}

// Create handles POST /api/projects/{projectID}/pipelines.
func (h *PipelineHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	// Confirm the project belongs to the caller's tenant before creating a
	// pipeline under it — otherwise any authenticated user could create a
	// pipeline in another tenant's project by guessing its UUID.
	tenantID := middleware.TenantIDFromContext(r.Context())
	if _, err := h.projects.Get(r.Context(), projectID, tenantID); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var req dto.CreatePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	pipeline, err := h.pipelines.Create(r.Context(), domain.Pipeline{
		ProjectID:   projectID,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create pipeline")
		return
	}

	writeJSON(w, http.StatusCreated, dto.PipelineFromDomain(pipeline))
}

// ListByProject handles GET /api/projects/{projectID}/pipelines.
func (h *PipelineHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())
	if _, err := h.projects.Get(r.Context(), projectID, tenantID); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	pipelines, err := h.pipelines.ListByProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pipelines")
		return
	}

	writeJSON(w, http.StatusOK, dto.PipelinesFromDomain(pipelines))
}

// Get handles GET /api/pipelines/{id}.
func (h *PipelineHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())
	pipeline, err := h.pipelines.Get(r.Context(), id, tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}

	writeJSON(w, http.StatusOK, dto.PipelineFromDomain(pipeline))
}

// Update handles PUT /api/pipelines/{id}.
func (h *PipelineHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	var req dto.UpdatePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())
	if err := h.pipelines.Update(r.Context(), domain.Pipeline{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
	}, tenantID); err != nil {
		// Update returns "not found" when the row doesn't match id+tenant.
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}

	pipeline, err := h.pipelines.Get(r.Context(), id, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get updated pipeline")
		return
	}

	writeJSON(w, http.StatusOK, dto.PipelineFromDomain(pipeline))
}

// Delete handles DELETE /api/pipelines/{id}.
func (h *PipelineHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())
	if err := h.pipelines.Delete(r.Context(), id, tenantID); err != nil {
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SaveVersion handles POST /api/pipelines/{id}/versions.
func (h *PipelineHandler) SaveVersion(w http.ResponseWriter, r *http.Request) {
	pipelineID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	if !h.assertPipelineOwned(w, r, pipelineID) {
		return
	}

	var req dto.SaveVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Graph.Nodes == nil {
		writeError(w, http.StatusBadRequest, "graph with at least a nodes array is required")
		return
	}

	// Determine next version number
	nextVersion := 1
	latest, err := h.versions.GetLatest(r.Context(), pipelineID)
	if err == nil {
		nextVersion = latest.Version + 1
	}

	pv, err := h.versions.Create(r.Context(), domain.PipelineVersion{
		PipelineID: pipelineID,
		Version:    nextVersion,
		Graph:      req.Graph,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save version")
		return
	}

	writeJSON(w, http.StatusCreated, dto.PipelineVersionFromDomain(pv))
}

// ListVersions handles GET /api/pipelines/{id}/versions.
func (h *PipelineHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	pipelineID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	if !h.assertPipelineOwned(w, r, pipelineID) {
		return
	}

	versions, err := h.versions.ListByPipeline(r.Context(), pipelineID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list versions")
		return
	}

	writeJSON(w, http.StatusOK, dto.PipelineVersionsFromDomain(versions))
}

// GetVersion handles GET /api/pipelines/{id}/versions/{version}.
func (h *PipelineHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	pipelineID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	if !h.assertPipelineOwned(w, r, pipelineID) {
		return
	}

	version, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid version number")
		return
	}

	pv, err := h.versions.GetByVersion(r.Context(), pipelineID, version)
	if err != nil {
		writeError(w, http.StatusNotFound, "version not found")
		return
	}

	writeJSON(w, http.StatusOK, dto.PipelineVersionFromDomain(pv))
}

// GetLatestVersion handles GET /api/pipelines/{id}/versions/latest.
func (h *PipelineHandler) GetLatestVersion(w http.ResponseWriter, r *http.Request) {
	pipelineID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	if !h.assertPipelineOwned(w, r, pipelineID) {
		return
	}

	pv, err := h.versions.GetLatest(r.Context(), pipelineID)
	if err != nil {
		writeError(w, http.StatusNotFound, "no versions found")
		return
	}

	writeJSON(w, http.StatusOK, dto.PipelineVersionFromDomain(pv))
}

// pipelineExport is the JSON shape for pipeline export/import.
type pipelineExport struct {
	Name          string               `json:"name"`
	Version       int                  `json:"version"`
	ClothoVersion string               `json:"clotho_version"`
	Graph         domain.PipelineGraph `json:"graph"`
}

// Export handles GET /api/pipelines/{id}/export.
func (h *PipelineHandler) Export(w http.ResponseWriter, r *http.Request) {
	pipelineID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())
	pipeline, err := h.pipelines.Get(r.Context(), pipelineID, tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}

	pv, err := h.versions.GetLatest(r.Context(), pipelineID)
	if err != nil {
		writeError(w, http.StatusNotFound, "no versions found")
		return
	}

	// Strip sensitive fields from node configs (credential IDs, tenant info).
	sanitizedNodes := make([]domain.NodeInstance, len(pv.Graph.Nodes))
	for i, node := range pv.Graph.Nodes {
		sanitizedNodes[i] = node
		sanitizedNodes[i].Config = stripSensitiveConfig(node.Config)
	}

	export := pipelineExport{
		Name:          pipeline.Name,
		Version:       pv.Version,
		ClothoVersion: "0.1.0",
		Graph: domain.PipelineGraph{
			Nodes:    sanitizedNodes,
			Edges:    pv.Graph.Edges,
			Viewport: pv.Graph.Viewport,
		},
	}

	filename := fmt.Sprintf("%s.clotho.json", pipeline.Name)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	writeJSON(w, http.StatusOK, export)
}

// stripSensitiveConfig removes credential_id and tenant_id from a node config.
func stripSensitiveConfig(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw
	}
	delete(m, "credential_id")
	delete(m, "tenant_id")
	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return out
}

// importMaxNodes caps the node count on import. 200 comfortably holds every
// creative pipeline we've seen; more than that is either abuse or a bug.
const importMaxNodes = 200

// importMaxEdges caps the edge count independently; a fully connected 200-node
// graph would have ~20k edges, which is nonsense for a creative pipeline.
const importMaxEdges = 500

// Import handles POST /api/pipelines/{id}/import.
func (h *PipelineHandler) Import(w http.ResponseWriter, r *http.Request) {
	pipelineID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	if !h.assertPipelineOwned(w, r, pipelineID) {
		return
	}

	// Override the group's 1 MB default — import exports can be larger
	// but still bounded. 10 MB is the ceiling beyond which we assume abuse.
	r.Body = http.MaxBytesReader(w, r.Body, middleware.PipelineImportMaxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var imp pipelineExport
	if err := dec.Decode(&imp); err != nil {
		if middleware.WriteMaxBytesError(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate nodes have required fields.
	if len(imp.Graph.Nodes) == 0 {
		writeError(w, http.StatusBadRequest, "graph must contain at least one node")
		return
	}
	if len(imp.Graph.Nodes) > importMaxNodes {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("graph has too many nodes (%d, max %d)", len(imp.Graph.Nodes), importMaxNodes))
		return
	}
	if len(imp.Graph.Edges) > importMaxEdges {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("graph has too many edges (%d, max %d)", len(imp.Graph.Edges), importMaxEdges))
		return
	}
	nodeIDs := make(map[string]bool, len(imp.Graph.Nodes))
	for _, node := range imp.Graph.Nodes {
		if node.ID == "" {
			writeError(w, http.StatusBadRequest, "each node must have an id")
			return
		}
		if node.Type == "" {
			writeError(w, http.StatusBadRequest, "each node must have a type")
			return
		}
		nodeIDs[node.ID] = true
	}

	// Validate edges reference valid node IDs.
	for _, edge := range imp.Graph.Edges {
		if !nodeIDs[edge.Source] {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("edge references unknown source node %q", edge.Source))
			return
		}
		if !nodeIDs[edge.Target] {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("edge references unknown target node %q", edge.Target))
			return
		}
	}

	// Determine next version number.
	nextVersion := 1
	latest, err := h.versions.GetLatest(r.Context(), pipelineID)
	if err == nil {
		nextVersion = latest.Version + 1
	}

	pv, err := h.versions.Create(r.Context(), domain.PipelineVersion{
		PipelineID: pipelineID,
		Version:    nextVersion,
		Graph:      imp.Graph,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create imported version")
		return
	}

	writeJSON(w, http.StatusCreated, dto.PipelineVersionFromDomain(pv))
}
