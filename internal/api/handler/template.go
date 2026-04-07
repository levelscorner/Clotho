package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/user/clotho/internal/templates"
)

// TemplateHandler serves built-in pipeline templates.
type TemplateHandler struct{}

// NewTemplateHandler creates a TemplateHandler.
func NewTemplateHandler() *TemplateHandler {
	return &TemplateHandler{}
}

// Routes registers template routes on the given router.
func (h *TemplateHandler) Routes(r chi.Router) {
	r.Get("/api/templates", h.List)
	r.Get("/api/templates/{id}", h.Get)
}

// List handles GET /api/templates — returns summaries without graphs.
func (h *TemplateHandler) List(w http.ResponseWriter, _ *http.Request) {
	all := templates.All()
	summaries := make([]templates.TemplateSummary, len(all))
	for i, t := range all {
		summaries[i] = t.Summary()
	}
	writeJSON(w, http.StatusOK, summaries)
}

// Get handles GET /api/templates/{id} — returns a single template with graph.
func (h *TemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t := templates.ByID(id)
	if t == nil {
		writeError(w, http.StatusNotFound, "template not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}
