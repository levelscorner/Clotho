package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/engine"
)

// stubExecutor is a minimal StepExecutor used to drive node_test_handler
// scenarios. ScriptedOutput is returned on success; ScriptedErr short-
// circuits Execute.
type stubExecutor struct {
	output engine.StepOutput
	err    error
}

func (e *stubExecutor) Execute(_ context.Context, _ domain.NodeInstance, _ map[string]json.RawMessage) (engine.StepOutput, error) {
	return e.output, e.err
}

func (e *stubExecutor) ExecuteStream(_ context.Context, _ domain.NodeInstance, _ map[string]json.RawMessage) (<-chan engine.ExecutorStreamChunk, <-chan engine.StepOutput, <-chan error) {
	chunks := make(chan engine.ExecutorStreamChunk)
	close(chunks)
	res := make(chan engine.StepOutput, 1)
	errCh := make(chan error, 1)
	if e.err != nil {
		errCh <- e.err
	} else {
		res <- e.output
	}
	return chunks, res, errCh
}

func newTestNodeHandler(stub engine.StepExecutor, nt domain.NodeType) *NodeTestHandler {
	reg := engine.NewExecutorRegistry()
	reg.Register(nt, stub)
	return NewNodeTestHandler(reg)
}

func postNodeTest(t *testing.T, h *NodeTestHandler, body any) *httptest.ResponseRecorder {
	t.Helper()
	bs, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/nodes/test", bytes.NewReader(bs))
	rr := httptest.NewRecorder()
	h.Test(rr, req)
	return rr
}

func TestNodeTestHandler_HappyPathReturnsOutput(t *testing.T) {
	t.Parallel()
	tokens := 12
	cost := 0.0042
	stub := &stubExecutor{
		output: engine.StepOutput{
			Data:       json.RawMessage(`"hello world"`),
			TokensUsed: &tokens,
			CostUSD:    &cost,
		},
	}
	h := newTestNodeHandler(stub, domain.NodeTypeAgent)

	rr := postNodeTest(t, h, map[string]any{
		"node": map[string]any{
			"id":   "n1",
			"type": "agent",
		},
		"inputs": map[string]string{},
	})

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rr.Code, rr.Body.String())
	}
	var got struct {
		Output     json.RawMessage `json:"output"`
		TokensUsed *int            `json:"tokens_used"`
		CostUSD    *float64        `json:"cost_usd"`
		Failure    any             `json:"failure"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(got.Output) != `"hello world"` {
		t.Errorf("output = %s, want \"hello world\"", string(got.Output))
	}
	if got.TokensUsed == nil || *got.TokensUsed != 12 {
		t.Errorf("tokens_used = %v, want 12", got.TokensUsed)
	}
	if got.Failure != nil {
		t.Errorf("failure should be nil on success: %v", got.Failure)
	}
}

func TestNodeTestHandler_FailureReturnsStructuredFailure(t *testing.T) {
	t.Parallel()
	stub := &stubExecutor{err: errors.New("provider exploded")}
	h := newTestNodeHandler(stub, domain.NodeTypeAgent)

	rr := postNodeTest(t, h, map[string]any{
		"node": map[string]any{"id": "n1", "type": "agent"},
	})

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 even on step failure", rr.Code)
	}
	var got struct {
		Failure map[string]any `json:"failure"`
		Error   string         `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Failure == nil {
		t.Fatal("expected failure object in response")
	}
	if got.Failure["class"] == nil {
		t.Errorf("failure.class missing: %v", got.Failure)
	}
	if got.Error == "" {
		t.Errorf("error string should still be present for back-compat")
	}
}

func TestNodeTestHandler_MissingNodeTypeReturns400(t *testing.T) {
	t.Parallel()
	h := newTestNodeHandler(&stubExecutor{}, domain.NodeTypeAgent)

	rr := postNodeTest(t, h, map[string]any{
		"node": map[string]any{"id": "n1"}, // no type
	})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestNodeTestHandler_UnknownNodeTypeReturns400(t *testing.T) {
	t.Parallel()
	h := newTestNodeHandler(&stubExecutor{}, domain.NodeTypeAgent)

	rr := postNodeTest(t, h, map[string]any{
		"node": map[string]any{"id": "n1", "type": "alien"},
	})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (no executor for unknown type)", rr.Code)
	}
}

func TestNodeTestHandler_NoRegistryReturns503(t *testing.T) {
	t.Parallel()
	h := NewNodeTestHandler(nil)
	rr := postNodeTest(t, h, map[string]any{
		"node": map[string]any{"id": "n1", "type": "agent"},
	})
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rr.Code)
	}
}

func TestNodeTestHandler_BadJSONReturns400(t *testing.T) {
	t.Parallel()
	h := newTestNodeHandler(&stubExecutor{}, domain.NodeTypeAgent)
	req := httptest.NewRequest(http.MethodPost, "/api/nodes/test", bytes.NewReader([]byte(`not json`)))
	rr := httptest.NewRecorder()
	h.Test(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}
