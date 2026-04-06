package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/user/clotho/internal/engine"
)

// StreamHandler handles SSE streaming for execution events.
type StreamHandler struct {
	eventBus *engine.EventBus
}

// NewStreamHandler creates a StreamHandler.
func NewStreamHandler(eventBus *engine.EventBus) *StreamHandler {
	return &StreamHandler{eventBus: eventBus}
}

// Routes registers stream routes on the given router.
func (h *StreamHandler) Routes(r chi.Router) {
	r.Get("/api/executions/{id}/stream", h.Stream)
}

// Stream handles GET /api/executions/{id}/stream (SSE endpoint).
func (h *StreamHandler) Stream(w http.ResponseWriter, r *http.Request) {
	executionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid execution ID")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Subscribe to events for this execution
	ch := h.eventBus.Subscribe(executionID)
	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			h.eventBus.Unsubscribe(executionID, ch)
			return
		case event, ok := <-ch:
			if !ok {
				// Channel closed (execution done)
				return
			}

			data, err := json.Marshal(event)
			if err != nil {
				continue
			}

			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			flusher.Flush()

			// Close after terminal events
			if event.Type == engine.EventExecutionCompleted || event.Type == engine.EventExecutionFailed {
				h.eventBus.Unsubscribe(executionID, ch)
				return
			}
		}
	}
}
