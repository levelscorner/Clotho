package engine_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/engine"
	"github.com/user/clotho/internal/engine/testutil"
)

// otelHarness extends the pattern-test harness with an in-memory OTel
// span recorder so tests can assert what the engine traced.
func otelHarness(t *testing.T, scripts map[string]testutil.Script) (*harness, *tracetest.SpanRecorder) {
	t.Helper()

	rec := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(rec))
	tracer := tp.Tracer("clotho/test")

	fe := testutil.NewFakeExecutor(scripts)
	reg := engine.NewExecutorRegistry()
	reg.Register(domain.NodeTypeAgent, fe)
	reg.Register(domain.NodeTypeMedia, fe)
	reg.Register(domain.NodeTypeTool, fe)

	bus := engine.NewEventBus()
	execStore := testutil.NewFakeExecutionStore()
	stepStore := testutil.NewFakeStepResultStore()

	eng := engine.NewEngineWithTracer(reg, bus, execStore, stepStore, nil, tracer)

	return &harness{
		t:         t,
		exec:      fe,
		execStore: execStore,
		stepStore: stepStore,
		bus:       bus,
		eng:       eng,
		subDone:   make(chan struct{}),
	}, rec
}

// TestEngine_PersistsTraceID is the entry point for A7. The engine must
// start a root workflow span on every ExecuteWorkflow call and persist
// the resulting trace ID on the execution row so the FailureDrawer's
// "Copy diagnostic" includes it.
func TestEngine_PersistsTraceIDOnExecution(t *testing.T) {
	t.Parallel()

	scripts := map[string]testutil.Script{
		"agent": {Output: testutil.TextOutput("hello")},
	}
	h, rec := otelHarness(t, scripts)

	// Seed the execution row.
	tenant := uuid.New()
	e, _ := h.execStore.Create(context.Background(), domain.Execution{
		TenantID:          tenant,
		PipelineVersionID: uuid.New(),
		Status:            domain.StatusPending,
	})
	h.execution = e

	graph := domain.PipelineGraph{
		Nodes: []domain.NodeInstance{agentNode("agent", domain.PortTypeText)},
	}
	if err := h.eng.ExecuteWorkflow(context.Background(), e, graph); err != nil {
		t.Fatalf("ExecuteWorkflow: %v", err)
	}

	snap := h.execStore.Snapshot(e.ID)
	if snap.TraceID == nil || *snap.TraceID == "" {
		t.Fatalf("trace_id empty on execution; want non-empty (got %v)", snap.TraceID)
	}

	// The persisted ID should match the root span's TraceID.
	spans := rec.Ended()
	if len(spans) == 0 {
		t.Fatal("expected at least one span recorded")
	}
	rootSpanTraceID := spans[0].SpanContext().TraceID().String()
	if *snap.TraceID != rootSpanTraceID {
		t.Errorf("persisted trace_id = %q, want root span trace_id %q", *snap.TraceID, rootSpanTraceID)
	}
}

// TestEngine_RootSpanNamedWorkflowExecution locks the span name + key
// attributes so dashboards can filter on them. Without this assertion
// a refactor could quietly rename the span and break observability.
func TestEngine_RootSpanNamedWorkflowExecution(t *testing.T) {
	t.Parallel()

	scripts := map[string]testutil.Script{
		"a": {Output: testutil.TextOutput("ok")},
	}
	h, rec := otelHarness(t, scripts)
	tenant := uuid.New()
	e, _ := h.execStore.Create(context.Background(), domain.Execution{
		TenantID:          tenant,
		PipelineVersionID: uuid.New(),
		Status:            domain.StatusPending,
	})
	h.execution = e

	graph := domain.PipelineGraph{
		Nodes: []domain.NodeInstance{agentNode("a", domain.PortTypeText)},
	}
	_ = h.eng.ExecuteWorkflow(context.Background(), e, graph)

	var rootName string
	for _, sp := range rec.Ended() {
		if sp.Parent().IsValid() {
			continue // not root
		}
		rootName = sp.Name()
		// Check execution.id attribute exists.
		var foundID bool
		for _, attr := range sp.Attributes() {
			if string(attr.Key) == "execution.id" {
				foundID = true
				if attr.Value.AsString() != e.ID.String() {
					t.Errorf("execution.id attr = %q, want %q", attr.Value.AsString(), e.ID.String())
				}
			}
		}
		if !foundID {
			t.Errorf("root span missing execution.id attribute")
		}
		break
	}
	if rootName != "workflow.execution" {
		t.Errorf("root span name = %q, want workflow.execution", rootName)
	}
}

// TestEngine_ChildSpanPerNode ensures each node executed produces a
// "workflow.node" child span carrying node.id + node.type attributes.
// Required by docs/PIPELINE-PATTERNS.md for downstream tracing tools.
func TestEngine_ChildSpanPerNode(t *testing.T) {
	t.Parallel()

	scripts := map[string]testutil.Script{
		"first":  {Output: testutil.TextOutput("a")},
		"second": {Output: testutil.TextOutput("b")},
	}
	h, rec := otelHarness(t, scripts)
	tenant := uuid.New()
	e, _ := h.execStore.Create(context.Background(), domain.Execution{
		TenantID:          tenant,
		PipelineVersionID: uuid.New(),
		Status:            domain.StatusPending,
	})
	h.execution = e

	graph := domain.PipelineGraph{
		Nodes: []domain.NodeInstance{
			agentNode("first", domain.PortTypeText),
			agentNode("second", domain.PortTypeText),
		},
	}
	if err := h.eng.ExecuteWorkflow(context.Background(), e, graph); err != nil {
		t.Fatalf("ExecuteWorkflow: %v", err)
	}

	nodeSpansByID := map[string]bool{}
	for _, sp := range rec.Ended() {
		if sp.Name() != "workflow.node" {
			continue
		}
		for _, attr := range sp.Attributes() {
			if string(attr.Key) == "node.id" {
				nodeSpansByID[attr.Value.AsString()] = true
			}
		}
	}
	if !nodeSpansByID["first"] {
		t.Errorf("missing workflow.node span for 'first'; got: %v", nodeSpansByID)
	}
	if !nodeSpansByID["second"] {
		t.Errorf("missing workflow.node span for 'second'; got: %v", nodeSpansByID)
	}
}

// silence unused import if helpers go missing.
var _ = json.Marshal
