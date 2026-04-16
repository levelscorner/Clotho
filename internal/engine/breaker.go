package engine

import (
	"errors"
	"sync"
	"time"

	"github.com/user/clotho/internal/domain"
)

// ErrCircuitOpen is the sentinel returned by Breaker.Allow when a
// (provider, model) key has tripped its threshold and is in cooldown.
// Wrap or rely on errors.Is for checks; ClassifyProviderError converts
// it into a domain.FailureCircuitOpen StepFailure for the UI.
var ErrCircuitOpen = errors.New("circuit breaker open")

// BreakerState matches the four-state model in docs/research1.md.
//
//   Closed   — normal traffic; all requests pass.
//   Degraded — failure rate elevated but below trip threshold; traffic
//              still passes but the state is observable for alerting.
//   Open     — trip threshold reached; all requests short-circuit with
//              ErrCircuitOpen until the cooldown elapses.
//   HalfOpen — cooldown elapsed; allow a small number of probe requests
//              to test recovery. A success closes the breaker, a failure
//              re-opens it (with the same cooldown).
type BreakerState string

const (
	StateClosed   BreakerState = "closed"
	StateDegraded BreakerState = "degraded"
	StateOpen     BreakerState = "open"
	StateHalfOpen BreakerState = "half_open"
)

// BreakerConfig tunes the thresholds. Defaults via DefaultBreakerConfig
// match research1.md and are conservative — easy to retune later from a
// single point.
type BreakerConfig struct {
	// Window over which failures are counted.
	Window time.Duration
	// Failure count within Window that flips Closed → Degraded.
	DegradedThreshold int
	// Failure count within Window that flips Closed/Degraded → Open.
	OpenThreshold int
	// How long to stay Open before transitioning to HalfOpen.
	Cooldown time.Duration
	// Number of probe requests allowed in HalfOpen before deciding.
	HalfOpenProbes int
	// Clock function — overridden in tests for deterministic windows.
	Now func() time.Time
}

// DefaultBreakerConfig returns conservative settings: 5 failures in 60s
// trips the breaker, 3 failures puts it in Degraded, 30s cooldown,
// 3 probes in HalfOpen.
func DefaultBreakerConfig() BreakerConfig {
	return BreakerConfig{
		Window:            60 * time.Second,
		DegradedThreshold: 3,
		OpenThreshold:     5,
		Cooldown:          30 * time.Second,
		HalfOpenProbes:    3,
		Now:               time.Now,
	}
}

// Breaker is a per-key circuit breaker. Goroutine-safe. Failures and
// successes contribute to a sliding window of timestamps; the state is
// derived from that window plus the elapsed-since-trip clock.
type Breaker struct {
	cfg BreakerConfig

	mu               sync.Mutex
	state            BreakerState
	failures         []time.Time // timestamps within Window
	openedAt         time.Time
	halfOpenInFlight int
	halfOpenSuccess  int
}

// NewBreaker constructs a Breaker with the provided config. If cfg.Now
// is nil, time.Now is used.
func NewBreaker(cfg BreakerConfig) *Breaker {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Breaker{cfg: cfg, state: StateClosed}
}

// State returns the current breaker state, after dropping any expired
// failure entries from the sliding window. Caller-visible state may
// differ from the stored state when a cooldown has elapsed (Open →
// HalfOpen on the next call).
func (b *Breaker) State() BreakerState {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tickLocked()
	return b.state
}

// Allow reports whether a new request may proceed. Returns nil if it
// may, ErrCircuitOpen if not. In HalfOpen state, only HalfOpenProbes
// concurrent calls are admitted; further calls also receive
// ErrCircuitOpen.
func (b *Breaker) Allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tickLocked()

	switch b.state {
	case StateOpen:
		return ErrCircuitOpen
	case StateHalfOpen:
		if b.halfOpenInFlight >= b.cfg.HalfOpenProbes {
			return ErrCircuitOpen
		}
		b.halfOpenInFlight++
	}
	return nil
}

// RecordSuccess marks a successful call. In HalfOpen, accumulating
// HalfOpenProbes successes closes the breaker; in any other state the
// failure window is left to expire naturally.
func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tickLocked()

	if b.state == StateHalfOpen {
		b.halfOpenInFlight--
		b.halfOpenSuccess++
		if b.halfOpenSuccess >= b.cfg.HalfOpenProbes {
			b.resetLocked()
		}
		return
	}

	// In Closed/Degraded, a single success doesn't clear the window —
	// failures expire on their own. This avoids "one good call masks ten
	// bad ones" oscillation on flaky upstreams.
}

// RecordFailure marks a failed call. Adds to the sliding window; if the
// count crosses thresholds the state moves accordingly. In HalfOpen,
// any failure immediately re-opens with the original cooldown.
func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tickLocked()

	now := b.cfg.Now()

	if b.state == StateHalfOpen {
		b.halfOpenInFlight--
		b.openLocked(now)
		return
	}

	b.failures = append(b.failures, now)
	count := len(b.failures)

	switch {
	case count >= b.cfg.OpenThreshold:
		b.openLocked(now)
	case count >= b.cfg.DegradedThreshold:
		b.state = StateDegraded
	}
}

// tickLocked drops expired failures and may move Open → HalfOpen if the
// cooldown has elapsed. Caller must hold b.mu.
func (b *Breaker) tickLocked() {
	now := b.cfg.Now()
	cutoff := now.Add(-b.cfg.Window)

	// Drop expired failures.
	if len(b.failures) > 0 {
		i := 0
		for i < len(b.failures) && b.failures[i].Before(cutoff) {
			i++
		}
		if i > 0 {
			b.failures = b.failures[i:]
		}
	}

	// Cooldown elapsed → HalfOpen.
	if b.state == StateOpen && now.Sub(b.openedAt) >= b.cfg.Cooldown {
		b.state = StateHalfOpen
		b.halfOpenInFlight = 0
		b.halfOpenSuccess = 0
	}

	// Window emptied while Degraded → back to Closed.
	if b.state == StateDegraded && len(b.failures) == 0 {
		b.state = StateClosed
	}
}

func (b *Breaker) openLocked(now time.Time) {
	b.state = StateOpen
	b.openedAt = now
	b.halfOpenInFlight = 0
	b.halfOpenSuccess = 0
}

func (b *Breaker) resetLocked() {
	b.state = StateClosed
	b.failures = nil
	b.openedAt = time.Time{}
	b.halfOpenInFlight = 0
	b.halfOpenSuccess = 0
}

// BreakerRegistry hands out per-key Breakers. Key convention is
// "{provider}:{model}" so distinct models don't punish each other when
// one of them flakes. Goroutine-safe; lazily creates a new Breaker per
// key on first lookup.
type BreakerRegistry struct {
	cfg      BreakerConfig
	mu       sync.Mutex
	breakers map[string]*Breaker
}

// NewBreakerRegistry builds a registry whose new Breakers all share the
// same config. Pass DefaultBreakerConfig() to use the documented values.
func NewBreakerRegistry(cfg BreakerConfig) *BreakerRegistry {
	return &BreakerRegistry{cfg: cfg, breakers: make(map[string]*Breaker)}
}

// For returns the Breaker for the given (provider, model) key, creating
// one if it doesn't exist. Empty provider/model collapses to a single
// shared "unknown" key — better than panicking, but caller should always
// pass real values.
func (r *BreakerRegistry) For(provider, model string) *Breaker {
	key := provider + ":" + model
	if provider == "" && model == "" {
		key = "unknown"
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.breakers[key]
	if !ok {
		b = NewBreaker(r.cfg)
		r.breakers[key] = b
	}
	return b
}

// circuitOpenFailure builds the StepFailure surfaced when Allow returns
// ErrCircuitOpen. Centralized here so the message and hint stay
// consistent between the agent executor and any future executor that
// wraps the same breaker.
func circuitOpenFailure(provider, model string, until time.Duration) domain.StepFailure {
	cooldownText := "cooling down"
	if until > 0 {
		cooldownText = "cooldown " + until.Round(time.Second).String()
	}
	return domain.StepFailure{
		Class:     domain.FailureCircuitOpen,
		Stage:     domain.StageProviderCall,
		Provider:  provider,
		Model:     model,
		Retryable: false,
		Message:   "Circuit breaker open for " + provider + ":" + model + " (" + cooldownText + ").",
		Hint:      hintFor[domain.FailureCircuitOpen],
		Attempts:  0,
		At:        time.Now().UTC(),
	}
}
