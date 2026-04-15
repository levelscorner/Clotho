package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/user/clotho/internal/api/middleware"
	"github.com/user/clotho/internal/engine"
	"github.com/user/clotho/internal/store"
)

// StreamHandler handles SSE streaming for execution events.
//
// SSE has no preflight, so CORS doesn't gate the initial connect the way it
// gates a fetch. This handler therefore does two extra checks on top of the
// auth middleware before calling Subscribe:
//
//  1. Tenant ownership — the authenticated caller's tenant must match the
//     execution row's tenant. Prevents cross-tenant stream reads.
//  2. Origin check — if the Origin header is present, it must be in the
//     configured allowlist. Mitigates cross-origin EventSource hijack
//     attempts from a page the user visits in the same browser session.
type StreamHandler struct {
	executions     store.ExecutionStore
	eventBus       *engine.EventBus
	allowedOrigins map[string]struct{}
}

// NewStreamHandler creates a StreamHandler.
func NewStreamHandler(executions store.ExecutionStore, eventBus *engine.EventBus, allowedOrigins []string) *StreamHandler {
	set := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o == "" {
			continue
		}
		set[o] = struct{}{}
	}
	return &StreamHandler{
		executions:     executions,
		eventBus:       eventBus,
		allowedOrigins: set,
	}
}

// Routes registers stream routes on the given router.
func (h *StreamHandler) Routes(r chi.Router) {
	r.Get("/api/executions/{id}/stream", h.Stream)
}

// isOriginAllowed returns true when the request's Origin is absent (same-origin)
// or present and in the allowlist. An empty allowlist means "allow all" to keep
// the dev bypass path simple; production configs should set AllowedOrigins.
func (h *StreamHandler) isOriginAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // same-origin request
	}
	if len(h.allowedOrigins) == 0 {
		return true
	}
	// Normalize — strip trailing slash if any.
	if u, err := url.Parse(origin); err == nil && u != nil && u.Host != "" {
		origin = u.Scheme + "://" + u.Host
	}
	_, ok := h.allowedOrigins[origin]
	return ok
}

// Stream handles GET /api/executions/{id}/stream (SSE endpoint).
func (h *StreamHandler) Stream(w http.ResponseWriter, r *http.Request) {
	executionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid execution ID")
		return
	}

	if !h.isOriginAllowed(r) {
		writeError(w, http.StatusForbidden, "origin not allowed")
		return
	}

	// Tenant isolation — stream only the caller's own executions.
	tenantID := middleware.TenantIDFromContext(r.Context())
	if _, err := h.executions.Get(r.Context(), executionID, tenantID); err != nil {
		writeError(w, http.StatusNotFound, "execution not found")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Subscribe to events for this execution.
	ch := h.eventBus.Subscribe(executionID)
	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			// Client disconnected.
			h.eventBus.Unsubscribe(executionID, ch)
			return
		case event, ok := <-ch:
			if !ok {
				// Channel closed (execution done).
				return
			}

			data, err := json.Marshal(event)
			if err != nil {
				continue
			}

			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			flusher.Flush()

			// Close after terminal events.
			if event.Type == engine.EventExecutionCompleted || event.Type == engine.EventExecutionFailed {
				h.eventBus.Unsubscribe(executionID, ch)
				return
			}
		}
	}
}
