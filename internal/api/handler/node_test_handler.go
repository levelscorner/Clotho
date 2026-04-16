package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/engine"
)

// NodeTestHandler exposes POST /api/nodes/test — runs a single node
// without creating an execution record. Cuts the iteration loop from
// "build whole pipeline → run → wait → see error" to "click Test, see
// result in seconds".
type NodeTestHandler struct {
	executors *engine.ExecutorRegistry
}

// NewNodeTestHandler returns a handler bound to the executor registry.
func NewNodeTestHandler(executors *engine.ExecutorRegistry) *NodeTestHandler {
	return &NodeTestHandler{executors: executors}
}

// Routes registers POST /api/nodes/test on the given router.
func (h *NodeTestHandler) Routes(r chi.Router) {
	r.Post("/api/nodes/test", h.Test)
}

// nodeTestRequest is the request body. The full NodeInstance lets the
// caller test exactly the node they're configuring without round-tripping
// through pipeline save. Inputs map upstream port IDs to JSON values; for
// most test scenarios the user supplies a single string.
type nodeTestRequest struct {
	Node   domain.NodeInstance        `json:"node"`
	Inputs map[string]json.RawMessage `json:"inputs"`
}

// nodeTestResponse mirrors the success path of step execution. Failure
// is the structured StepFailure so the FailureDrawer can render it.
// HTTP status stays 200 even on step failure — the front end checks
// `failure` and `error` fields like it would on a real run.
type nodeTestResponse struct {
	Output     json.RawMessage `json:"output,omitempty"`
	TokensUsed *int            `json:"tokens_used,omitempty"`
	CostUSD    *float64        `json:"cost_usd,omitempty"`
	DurationMs int64           `json:"duration_ms"`
	Failure    any             `json:"failure,omitempty"`
	Error      string          `json:"error,omitempty"`
}

// Test runs the StepExecutor for the given node + inputs. Wraps the call
// in a 60-second context timeout so a misconfigured node can't hang the
// HTTP handler indefinitely. No DB writes — this is a sandboxed call.
func (h *NodeTestHandler) Test(w http.ResponseWriter, r *http.Request) {
	if h.executors == nil {
		writeError(w, http.StatusServiceUnavailable, "executor registry not configured")
		return
	}

	var req nodeTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Node.Type == "" {
		writeError(w, http.StatusBadRequest, "node.type is required")
		return
	}

	executor, err := h.executors.Get(req.Node.Type)
	if err != nil {
		writeError(w, http.StatusBadRequest, "no executor for node type: "+string(req.Node.Type))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	start := time.Now()
	out, execErr := executor.Execute(ctx, req.Node, req.Inputs)
	duration := time.Since(start).Milliseconds()

	if execErr != nil {
		// Recover the structured StepFailure when the executor returned
		// one (output validation, circuit-open). Fall back to
		// classifying the raw error message so we always produce a
		// rich payload for the FailureDrawer.
		failure := engine.ClassifyExecutionError(execErr, "", "")
		writeJSON(w, http.StatusOK, nodeTestResponse{
			DurationMs: duration,
			Failure:    failure,
			Error:      execErr.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, nodeTestResponse{
		Output:     out.Data,
		TokensUsed: out.TokensUsed,
		CostUSD:    out.CostUSD,
		DurationMs: duration,
	})
}
