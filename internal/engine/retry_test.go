package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryWithBackoff_SucceedsFirstTry(t *testing.T) {
	t.Parallel()
	policy := RetryPolicy{Attempts: 3, InitialBackoff: time.Millisecond}
	attempts, err := retryWithBackoff(context.Background(), policy,
		func(error) bool { return true },
		func(ctx context.Context, n int) error { return nil },
	)
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetryWithBackoff_RetriesUntilSuccess(t *testing.T) {
	t.Parallel()
	policy := RetryPolicy{
		Attempts:       3,
		InitialBackoff: time.Microsecond,
		BackoffFactor:  2.0,
		MaxBackoff:     time.Millisecond,
	}
	calls := 0
	attempts, err := retryWithBackoff(context.Background(), policy,
		func(error) bool { return true },
		func(ctx context.Context, n int) error {
			calls++
			if calls < 3 {
				return errors.New("flaky")
			}
			return nil
		},
	)
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetryWithBackoff_StopsOnNonRetryable(t *testing.T) {
	t.Parallel()
	policy := RetryPolicy{Attempts: 5, InitialBackoff: time.Microsecond}
	calls := 0
	wantErr := errors.New("auth failed")
	attempts, err := retryWithBackoff(context.Background(), policy,
		func(e error) bool { return false },
		func(ctx context.Context, n int) error {
			calls++
			return wantErr
		},
	)
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (non-retryable)", attempts)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestRetryWithBackoff_GivesUpAfterAttempts(t *testing.T) {
	t.Parallel()
	policy := RetryPolicy{Attempts: 3, InitialBackoff: time.Microsecond}
	calls := 0
	_, err := retryWithBackoff(context.Background(), policy,
		func(error) bool { return true },
		func(ctx context.Context, n int) error {
			calls++
			return errors.New("always fails")
		},
	)
	if err == nil {
		t.Errorf("expected err, got nil")
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestRetryWithBackoff_RespectsContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	policy := RetryPolicy{Attempts: 5, InitialBackoff: 100 * time.Millisecond}
	cancel() // cancel before first attempt

	_, err := retryWithBackoff(ctx, policy,
		func(error) bool { return true },
		func(ctx context.Context, n int) error { return errors.New("x") },
	)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}
