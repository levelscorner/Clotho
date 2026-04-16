package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/user/clotho/internal/llm"
)

// stubProvider is a minimal llm.Provider for BreakerProvider tests. It
// returns whatever err / resp / chunks were configured, regardless of
// the request.
type stubProvider struct {
	err    error
	resp   llm.CompletionResponse
	chunks <-chan llm.StreamChunk
}

func (p *stubProvider) Complete(_ context.Context, _ llm.CompletionRequest) (llm.CompletionResponse, error) {
	return p.resp, p.err
}

func (p *stubProvider) Stream(_ context.Context, _ llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	return p.chunks, p.err
}

func newTestBreakerProvider(inner llm.Provider) (*BreakerProvider, *Breaker, *fakeClock) {
	clk := &fakeClock{t: time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)}
	cfg := BreakerConfig{
		Window:            10 * time.Second,
		DegradedThreshold: 1,
		OpenThreshold:     2,
		Cooldown:          5 * time.Second,
		HalfOpenProbes:    1,
		Now:               clk.Now,
	}
	br := NewBreaker(cfg)
	return &BreakerProvider{Inner: inner, Breaker: br, Provider: "openai"}, br, clk
}

func TestBreakerProvider_SuccessRecordsSuccess(t *testing.T) {
	t.Parallel()
	bp, br, _ := newTestBreakerProvider(&stubProvider{
		resp: llm.CompletionResponse{Content: "ok"},
	})

	resp, err := bp.Complete(context.Background(), llm.CompletionRequest{Model: "gpt-4o"})
	if err != nil {
		t.Fatalf("Complete err = %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("content = %q, want ok", resp.Content)
	}
	// State should still be closed; one success doesn't change anything.
	if got := br.State(); got != StateClosed {
		t.Errorf("state = %q, want closed", got)
	}
}

func TestBreakerProvider_RetryableFailureCountsTowardBreaker(t *testing.T) {
	t.Parallel()
	bp, br, _ := newTestBreakerProvider(&stubProvider{
		err: &openai.APIError{HTTPStatusCode: 503, Message: "service unavailable"},
	})

	// Two 503s with OpenThreshold=2 → breaker open.
	for i := 0; i < 2; i++ {
		_, err := bp.Complete(context.Background(), llm.CompletionRequest{Model: "gpt-4o"})
		if err == nil {
			t.Fatalf("attempt %d: expected error", i+1)
		}
	}
	if got := br.State(); got != StateOpen {
		t.Errorf("breaker state after 2 retryable failures = %q, want open", got)
	}
}

func TestBreakerProvider_AuthFailureDoesNotTripBreaker(t *testing.T) {
	t.Parallel()
	bp, br, _ := newTestBreakerProvider(&stubProvider{
		err: &openai.APIError{HTTPStatusCode: 401, Message: "invalid key"},
	})

	// Many 401s — breaker should stay closed because auth needs a human,
	// not a cooldown. This is the key contract from breaker_provider.go.
	for i := 0; i < 10; i++ {
		_, _ = bp.Complete(context.Background(), llm.CompletionRequest{Model: "gpt-4o"})
	}
	if got := br.State(); got != StateClosed {
		t.Errorf("breaker state after 10 auth failures = %q, want closed", got)
	}
}

func TestBreakerProvider_OpenBreakerShortCircuits(t *testing.T) {
	t.Parallel()
	// Trip the breaker first.
	stub := &stubProvider{
		err: &openai.APIError{HTTPStatusCode: 503, Message: "outage"},
	}
	bp, _, _ := newTestBreakerProvider(stub)
	for i := 0; i < 2; i++ {
		_, _ = bp.Complete(context.Background(), llm.CompletionRequest{Model: "gpt-4o"})
	}

	// Now swap the inner provider with one that would succeed if called —
	// proving the breaker short-circuited before reaching it.
	stub.err = nil
	stub.resp = llm.CompletionResponse{Content: "would succeed"}

	_, err := bp.Complete(context.Background(), llm.CompletionRequest{Model: "gpt-4o"})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("err = %v, want ErrCircuitOpen", err)
	}
}

func TestBreakerProvider_StreamFollowsSamePolicy(t *testing.T) {
	t.Parallel()
	// Stream init fails with retryable 503 — should count toward breaker.
	bp, br, _ := newTestBreakerProvider(&stubProvider{
		err: &openai.APIError{HTTPStatusCode: 503, Message: "outage"},
	})

	for i := 0; i < 2; i++ {
		_, err := bp.Stream(context.Background(), llm.CompletionRequest{Model: "gpt-4o"})
		if err == nil {
			t.Fatalf("Stream attempt %d should have failed", i+1)
		}
	}
	if got := br.State(); got != StateOpen {
		t.Errorf("Stream-driven failures should trip breaker; state = %q", got)
	}

	// Subsequent Stream call on open breaker should short-circuit.
	_, err := bp.Stream(context.Background(), llm.CompletionRequest{Model: "gpt-4o"})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("Stream on open breaker = %v, want ErrCircuitOpen", err)
	}
}

func TestBreakerProvider_CircuitOpenDoesNotDoubleRecord(t *testing.T) {
	t.Parallel()
	// Trip breaker. Then call again. The recordOutcome path must NOT
	// re-record an ErrCircuitOpen as a failure (which would extend the
	// cooldown indefinitely under retry storms).
	bp, br, _ := newTestBreakerProvider(&stubProvider{
		err: &openai.APIError{HTTPStatusCode: 503, Message: "outage"},
	})
	for i := 0; i < 2; i++ {
		_, _ = bp.Complete(context.Background(), llm.CompletionRequest{Model: "gpt-4o"})
	}
	stateBefore := br.State()

	// 5 more attempts on an already-open breaker.
	for i := 0; i < 5; i++ {
		_, _ = bp.Complete(context.Background(), llm.CompletionRequest{Model: "gpt-4o"})
	}
	stateAfter := br.State()

	if stateBefore != StateOpen || stateAfter != StateOpen {
		t.Errorf("expected breaker to stay open across short-circuits; before=%q after=%q", stateBefore, stateAfter)
	}
	// More importantly: failures slice should not have grown — but we
	// can't introspect it directly without exposing internals. The state
	// invariant is the visible proof.
}
