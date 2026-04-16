package engine

import (
	"context"
	"errors"
	"time"

	"github.com/user/clotho/internal/llm"
)

// BreakerProvider wraps an llm.Provider with a circuit breaker keyed by
// (provider, model). It is the only path Phase A's reliability work
// touches — both Complete and Stream go through Allow / RecordSuccess /
// RecordFailure so the breaker sees every provider call regardless of
// streaming choice.
//
// Auth failures and other non-retryable classes do NOT count toward the
// breaker's failure window — those need a human, not a wait. Same for
// FailureValidation and FailureCostCap, which are entirely local.
type BreakerProvider struct {
	Inner    llm.Provider
	Breaker  *Breaker
	Provider string // for ClassifyProviderError + circuitOpenFailure messages
}

// Complete forwards the call only when the breaker permits, then folds
// the response/error back into the breaker's accounting.
func (b *BreakerProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	if err := b.Breaker.Allow(); err != nil {
		return llm.CompletionResponse{}, err
	}
	resp, err := b.Inner.Complete(ctx, req)
	b.recordOutcome(err, req.Model)
	return resp, err
}

// Stream forwards the call only when the breaker permits. We record
// success/failure on the *initial* error since the streaming goroutine
// runs after Stream returns; mid-stream failures are surfaced to the
// engine through the chunk channel, not through Stream's return value,
// so they're best handled by the caller wrapping with retry.
func (b *BreakerProvider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	if err := b.Breaker.Allow(); err != nil {
		return nil, err
	}
	ch, err := b.Inner.Stream(ctx, req)
	b.recordOutcome(err, req.Model)
	return ch, err
}

// recordOutcome interprets err through the classifier and only counts
// failures that should trip the breaker. ErrCircuitOpen passes through
// untouched (we don't re-record an already-open breaker).
func (b *BreakerProvider) recordOutcome(err error, model string) {
	if err == nil {
		b.Breaker.RecordSuccess()
		return
	}
	if errors.Is(err, ErrCircuitOpen) {
		return
	}
	failure := ClassifyProviderError(err, b.Provider, model)
	// Only retryable failures count against the breaker — auth and
	// validation failures need user intervention, not a cooldown.
	if failure.Retryable {
		b.Breaker.RecordFailure()
	} else {
		// A non-retryable failure that isn't the breaker's fault still
		// counts as "breaker did not stop a successful call" — record
		// success to avoid lingering Degraded state on a healthy upstream.
		b.Breaker.RecordSuccess()
	}
}

// breakerCooldownRemaining is a convenience used by error messages —
// reports how long until an Open breaker transitions to HalfOpen, or 0
// if not in Open state.
func breakerCooldownRemaining(b *Breaker) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state != StateOpen {
		return 0
	}
	elapsed := b.cfg.Now().Sub(b.openedAt)
	remaining := b.cfg.Cooldown - elapsed
	if remaining < 0 {
		return 0
	}
	return remaining
}
