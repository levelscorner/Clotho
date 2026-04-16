package engine

import (
	"context"
	"errors"
	"time"
)

// RetryPolicy controls retryWithBackoff behavior. ZeroValue is a no-op
// (Attempts<=1 means "try once, no retry").
type RetryPolicy struct {
	Attempts        int           // total attempts including the first
	InitialBackoff  time.Duration // wait before second attempt
	BackoffFactor   float64       // multiplier per attempt (e.g. 2.0)
	MaxBackoff      time.Duration // cap on per-attempt sleep
}

// DefaultRetryPolicy matches the plan: 3 attempts, 500ms initial, 2x
// factor, capped at 5s. Tuned for LLM provider calls where occasional
// 429/5xx is expected and a brief wait usually clears it.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		Attempts:       3,
		InitialBackoff: 500 * time.Millisecond,
		BackoffFactor:  2.0,
		MaxBackoff:     5 * time.Second,
	}
}

// retryWithBackoff invokes fn up to policy.Attempts times. After each
// failure, retryable(err) decides whether to retry; non-retryable
// failures or context cancellation short-circuit immediately. The
// returned attempt count is 1-based and reflects the attempt that
// produced the final error (or succeeded).
//
// Sleep happens via time.NewTimer so context cancellation interrupts
// the wait — important when an upstream cancels mid-backoff.
func retryWithBackoff(
	ctx context.Context,
	policy RetryPolicy,
	retryable func(err error) bool,
	fn func(ctx context.Context, attempt int) error,
) (attempts int, err error) {
	if policy.Attempts < 1 {
		policy.Attempts = 1
	}

	backoff := policy.InitialBackoff
	for attempt := 1; attempt <= policy.Attempts; attempt++ {
		// Honor cancellation before issuing the call.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return attempt, ctxErr
		}

		err = fn(ctx, attempt)
		if err == nil {
			return attempt, nil
		}

		// Last attempt → return whatever we have.
		if attempt == policy.Attempts {
			return attempt, err
		}

		// Decide retryability AFTER the call so we never sleep for an
		// auth failure that's never going to succeed.
		if retryable != nil && !retryable(err) {
			return attempt, err
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return attempt, err
		}

		// Sleep with cancellation support.
		t := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			t.Stop()
			return attempt, ctx.Err()
		case <-t.C:
		}

		// Grow backoff for the next attempt; cap at MaxBackoff.
		next := time.Duration(float64(backoff) * policy.BackoffFactor)
		if policy.MaxBackoff > 0 && next > policy.MaxBackoff {
			next = policy.MaxBackoff
		}
		backoff = next
	}

	return policy.Attempts, err
}
