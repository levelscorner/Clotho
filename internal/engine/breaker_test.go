package engine

import (
	"errors"
	"testing"
	"time"
)

// fakeClock yields a controllable time source for breaker tests.
type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time     { return c.t }
func (c *fakeClock) Advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestBreaker() (*Breaker, *fakeClock) {
	clk := &fakeClock{t: time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)}
	cfg := BreakerConfig{
		Window:            10 * time.Second,
		DegradedThreshold: 2,
		OpenThreshold:     3,
		Cooldown:          5 * time.Second,
		HalfOpenProbes:    2,
		Now:               clk.Now,
	}
	return NewBreaker(cfg), clk
}

func TestBreaker_StartsClosed(t *testing.T) {
	b, _ := newTestBreaker()
	if got := b.State(); got != StateClosed {
		t.Errorf("initial state = %q, want closed", got)
	}
	if err := b.Allow(); err != nil {
		t.Errorf("Allow on fresh breaker should be nil, got %v", err)
	}
}

func TestBreaker_DegradedAtThreshold(t *testing.T) {
	b, _ := newTestBreaker()
	b.RecordFailure()
	if got := b.State(); got != StateClosed {
		t.Errorf("after 1 failure: state = %q, want closed", got)
	}
	b.RecordFailure()
	if got := b.State(); got != StateDegraded {
		t.Errorf("after 2 failures (DegradedThreshold): state = %q, want degraded", got)
	}
	// Allow still passes in degraded.
	if err := b.Allow(); err != nil {
		t.Errorf("Allow in degraded state should be nil, got %v", err)
	}
}

func TestBreaker_OpensAtThreshold(t *testing.T) {
	b, _ := newTestBreaker()
	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}
	if got := b.State(); got != StateOpen {
		t.Errorf("after 3 failures (OpenThreshold): state = %q, want open", got)
	}
	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("Allow in open state should return ErrCircuitOpen, got %v", err)
	}
}

func TestBreaker_CooldownToHalfOpen(t *testing.T) {
	b, clk := newTestBreaker()
	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}
	if got := b.State(); got != StateOpen {
		t.Fatalf("expected open, got %q", got)
	}

	clk.Advance(4 * time.Second) // still in cooldown
	if got := b.State(); got != StateOpen {
		t.Errorf("before cooldown elapses: state = %q, want open", got)
	}

	clk.Advance(2 * time.Second) // total 6s > 5s cooldown
	if got := b.State(); got != StateHalfOpen {
		t.Errorf("after cooldown elapses: state = %q, want half_open", got)
	}
}

func TestBreaker_HalfOpenSuccessClosesBreaker(t *testing.T) {
	b, clk := newTestBreaker()
	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}
	clk.Advance(6 * time.Second)
	_ = b.State() // tick into half_open

	// Two successful probes (HalfOpenProbes=2) → Closed.
	if err := b.Allow(); err != nil {
		t.Fatalf("first probe Allow = %v", err)
	}
	b.RecordSuccess()
	if err := b.Allow(); err != nil {
		t.Fatalf("second probe Allow = %v", err)
	}
	b.RecordSuccess()
	if got := b.State(); got != StateClosed {
		t.Errorf("after HalfOpenProbes successes: state = %q, want closed", got)
	}
}

func TestBreaker_HalfOpenFailureReopens(t *testing.T) {
	b, clk := newTestBreaker()
	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}
	clk.Advance(6 * time.Second)
	_ = b.State()

	if err := b.Allow(); err != nil {
		t.Fatalf("probe Allow = %v", err)
	}
	b.RecordFailure()

	if got := b.State(); got != StateOpen {
		t.Errorf("after probe failure: state = %q, want open", got)
	}
}

func TestBreaker_HalfOpenLimitsConcurrentProbes(t *testing.T) {
	b, clk := newTestBreaker()
	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}
	clk.Advance(6 * time.Second)
	_ = b.State()

	// HalfOpenProbes = 2 — first two pass, third should be rejected.
	if err := b.Allow(); err != nil {
		t.Fatalf("probe 1: %v", err)
	}
	if err := b.Allow(); err != nil {
		t.Fatalf("probe 2: %v", err)
	}
	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("probe 3 should be rejected, got %v", err)
	}
}

func TestBreaker_FailuresExpireFromWindow(t *testing.T) {
	b, clk := newTestBreaker()
	b.RecordFailure()
	b.RecordFailure() // → degraded

	clk.Advance(11 * time.Second) // > Window=10s — both failures expire
	if got := b.State(); got != StateClosed {
		t.Errorf("after window elapses: state = %q, want closed", got)
	}
}

func TestBreaker_AuthFailureShouldNotTrip_byCaller(t *testing.T) {
	// The breaker only sees what the caller records. The caller
	// (BreakerProvider) decides not to record auth failures. This test
	// documents that contract by NOT calling RecordFailure for auth.
	b, _ := newTestBreaker()

	for i := 0; i < 10; i++ {
		// Caller would record success because auth errors don't trip.
		// In production the BreakerProvider just bypasses the call.
		b.RecordSuccess()
	}
	if got := b.State(); got != StateClosed {
		t.Errorf("breaker should remain closed: %q", got)
	}
}

func TestBreakerRegistry_KeysIndependently(t *testing.T) {
	r := NewBreakerRegistry(DefaultBreakerConfig())

	a := r.For("openai", "gpt-4o")
	bSame := r.For("openai", "gpt-4o")
	bDiff := r.For("openai", "gpt-4o-mini")
	cDiff := r.For("ollama", "gpt-4o")

	if a != bSame {
		t.Errorf("same key should return same Breaker instance")
	}
	if a == bDiff || a == cDiff {
		t.Errorf("different keys must return different Breakers")
	}
}
