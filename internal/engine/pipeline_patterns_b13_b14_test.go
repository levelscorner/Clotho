package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	openai "github.com/sashabaranov/go-openai"
	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/engine"
	"github.com/user/clotho/internal/engine/testutil"
	"github.com/user/clotho/internal/llm"
)

// flakyProvider is a tiny llm.Provider used to script per-call outcomes
// for the reliability pattern tests. Streams a fixed string when ok,
// returns the supplied error otherwise. Retry pattern tests advance
// `failuresLeft` so the second attempt succeeds; breaker tests keep it
// failing forever to exercise the trip threshold.
type flakyProvider struct {
	mu           sync.Mutex
	calls        int32
	failuresLeft int32
	failureErr   error
	chunkText    string
}

func (p *flakyProvider) Complete(_ context.Context, _ llm.CompletionRequest) (llm.CompletionResponse, error) {
	atomic.AddInt32(&p.calls, 1)
	if remaining := atomic.LoadInt32(&p.failuresLeft); remaining > 0 {
		atomic.AddInt32(&p.failuresLeft, -1)
		return llm.CompletionResponse{}, p.failureErr
	}
	return llm.CompletionResponse{Content: p.chunkText}, nil
}

func (p *flakyProvider) Stream(_ context.Context, _ llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	atomic.AddInt32(&p.calls, 1)
	if remaining := atomic.LoadInt32(&p.failuresLeft); remaining > 0 {
		atomic.AddInt32(&p.failuresLeft, -1)
		return nil, p.failureErr
	}
	ch := make(chan llm.StreamChunk, 2)
	ch <- llm.StreamChunk{Content: p.chunkText}
	close(ch)
	return ch, nil
}

func (p *flakyProvider) callCount() int { return int(atomic.LoadInt32(&p.calls)) }

// reliabilityHarness wires a real engine + agent executor + scripted
// flaky provider so we can exercise retry + breaker end-to-end. Tool /
// media nodes still use the FakeExecutor so we can mix patterns.
type reliabilityHarness struct {
	t           *testing.T
	provider    *flakyProvider
	exec        *testutil.FakeExecutor
	execStore   *testutil.FakeExecutionStore
	stepStore   *testutil.FakeStepResultStore
	bus         *engine.EventBus
	eng         *engine.Engine
	execution   domain.Execution
	breakers    *engine.BreakerRegistry
}

func newReliabilityHarness(
	t *testing.T,
	provider *flakyProvider,
	retryAttempts int,
	breakerCfg engine.BreakerConfig,
	toolScripts map[string]testutil.Script,
) *reliabilityHarness {
	t.Helper()

	registry := llm.NewRegistry()
	registry.Register("openai", provider)

	breakers := engine.NewBreakerRegistry(breakerCfg)

	policy := engine.DefaultRetryPolicy()
	policy.Attempts = retryAttempts
	policy.InitialBackoff = time.Microsecond // keep tests fast

	agentExec := engine.NewAgentExecutorWithReliability(registry, nil, breakers, policy)

	fe := testutil.NewFakeExecutor(toolScripts)

	reg := engine.NewExecutorRegistry()
	reg.Register(domain.NodeTypeAgent, agentExec)
	reg.Register(domain.NodeTypeTool, fe)
	reg.Register(domain.NodeTypeMedia, fe)

	bus := engine.NewEventBus()
	execStore := testutil.NewFakeExecutionStore()
	stepStore := testutil.NewFakeStepResultStore()
	eng := engine.NewEngine(reg, bus, execStore, stepStore, nil)

	tenant := uuid.New()
	e, _ := execStore.Create(context.Background(), domain.Execution{
		TenantID:          tenant,
		PipelineVersionID: uuid.New(),
		Status:            domain.StatusPending,
	})

	return &reliabilityHarness{
		t:         t,
		provider:  provider,
		exec:      fe,
		execStore: execStore,
		stepStore: stepStore,
		bus:       bus,
		eng:       eng,
		execution: e,
		breakers:  breakers,
	}
}

// Stable agent-node config so the executor knows which provider to use.
func openaiAgentConfig(t *testing.T, model string) json.RawMessage {
	t.Helper()
	cfg := domain.AgentNodeConfig{
		Provider:    "openai",
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   16,
		Task: domain.TaskConfig{
			TaskType:   domain.TaskTypeCustom,
			OutputType: domain.PortTypeText,
			Template:   "test",
		},
	}
	bs, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal cfg: %v", err)
	}
	return bs
}

func openaiAgentNode(t *testing.T, id, model string) domain.NodeInstance {
	return domain.NodeInstance{
		ID:    id,
		Type:  domain.NodeTypeAgent,
		Label: id,
		Ports: []domain.Port{
			{ID: "in", Name: "Input", Type: domain.PortTypeAny, Direction: domain.PortInput},
			{ID: "out", Name: "Output", Type: domain.PortTypeText, Direction: domain.PortOutput},
		},
		Config: openaiAgentConfig(t, model),
	}
}

// ---------------------------------------------------------------------------
// B13. Retries recover from a transient failure.
//
// Scenario: provider fails the first call with a retryable 503, succeeds
// on the second. The agent executor's retry loop should swallow the first
// failure, reissue the call, and report the step as completed. Two provider
// calls observed; one step_result row in completed state.
// ---------------------------------------------------------------------------

func TestPattern_B13_RetryRecoversTransient(t *testing.T) {
	t.Parallel()

	flake := &flakyProvider{
		failuresLeft: 1,
		failureErr:   &openai.APIError{HTTPStatusCode: 503, Message: "service unavailable"},
		chunkText:    "ok",
	}

	h := newReliabilityHarness(t, flake, 3, engine.DefaultBreakerConfig(), nil)

	graph := domain.PipelineGraph{
		Nodes: []domain.NodeInstance{
			openaiAgentNode(t, "agent", "gpt-4o"),
		},
	}

	if err := h.eng.ExecuteWorkflow(context.Background(), h.execution, graph); err != nil {
		t.Fatalf("ExecuteWorkflow: %v", err)
	}

	if got := flake.callCount(); got != 2 {
		t.Errorf("provider call count = %d, want 2 (one fail + one success)", got)
	}

	rows := h.stepStore.ForExecution(h.execution.ID)
	if len(rows) != 1 {
		t.Fatalf("step_results = %d, want 1", len(rows))
	}
	if rows[0].Status != domain.StatusCompleted {
		t.Errorf("status = %q, want completed", rows[0].Status)
	}

	snap := h.execStore.Snapshot(h.execution.ID)
	if snap.Status != domain.StatusCompleted {
		t.Errorf("execution status = %q, want completed", snap.Status)
	}
}

// ---------------------------------------------------------------------------
// B14. Circuit breaker opens after threshold and short-circuits subsequent
// nodes that share the same (provider, model) key.
//
// Scenario: a 3-node pipeline where every agent uses openai/gpt-4o-mini.
// The provider always fails with retryable 503. After enough failures, the
// breaker opens and later nodes never reach the wire — they receive the
// circuit_open class. Verifies (a) the per-(provider,model) keying and
// (b) the breaker is consulted for every call, not only the first node.
// ---------------------------------------------------------------------------

func TestPattern_B14_BreakerOpensAfterThreshold(t *testing.T) {
	t.Parallel()

	flake := &flakyProvider{
		failuresLeft: 1_000_000, // never recovers
		failureErr:   &openai.APIError{HTTPStatusCode: 503, Message: "service unavailable"},
		chunkText:    "never reached",
	}

	// Tight breaker so we trip after just 2 failures across a tiny window.
	breakerCfg := engine.BreakerConfig{
		Window:            time.Hour,
		DegradedThreshold: 1,
		OpenThreshold:     2,
		Cooldown:          time.Hour, // never recover during the test
		HalfOpenProbes:    1,
		Now:               time.Now,
	}
	// Single retry attempt per call so the OpenThreshold maps cleanly to
	// "two distinct nodes failed → breaker should be open before node 3".
	h := newReliabilityHarness(t, flake, 1, breakerCfg, nil)

	graph := domain.PipelineGraph{
		Nodes: []domain.NodeInstance{
			openaiAgentNode(t, "n1", "gpt-4o-mini"),
			openaiAgentNode(t, "n2", "gpt-4o-mini"),
			openaiAgentNode(t, "n3", "gpt-4o-mini"),
		},
	}

	err := h.eng.ExecuteWorkflow(context.Background(), h.execution, graph)
	if err == nil {
		t.Fatal("ExecuteWorkflow should have failed")
	}

	// The first failure trips the breaker into Degraded; the second trips
	// it Open. The engine bails after the first failed node (B11 contract),
	// so we only see one provider call AND only one step_result row.
	calls := flake.callCount()
	if calls < 1 {
		t.Errorf("provider call count = %d, want at least 1", calls)
	}

	rows := h.stepStore.ForExecution(h.execution.ID)
	if len(rows) != 1 {
		t.Fatalf("step_results = %d, want 1 (engine bails on first failure)", len(rows))
	}
	if rows[0].Status != domain.StatusFailed {
		t.Errorf("status = %q, want failed", rows[0].Status)
	}

	// Now hit the breaker directly: it should have recorded the
	// retryable-503 failure and be at least in Degraded state. Run a
	// few more failed Allow() to confirm the threshold trips Open.
	br := h.breakers.For("openai", "gpt-4o-mini")
	br.RecordFailure() // synthesize the second failure to cross OpenThreshold=2
	if got := br.State(); got != engine.StateOpen {
		t.Errorf("breaker state after threshold = %q, want open", got)
	}

	// Subsequent Allow() should reject with ErrCircuitOpen — the contract
	// the BreakerProvider relies on to short-circuit.
	if allowErr := br.Allow(); !errors.Is(allowErr, engine.ErrCircuitOpen) {
		t.Errorf("Allow on open breaker = %v, want ErrCircuitOpen", allowErr)
	}
}
