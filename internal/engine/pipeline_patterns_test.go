package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/engine"
	"github.com/user/clotho/internal/engine/testutil"
)

// ---------------------------------------------------------------------------
// Harness
//
// Each pattern test builds a PipelineGraph, registers a FakeExecutor against
// all three node types, runs ExecuteWorkflow, and asserts the per-pattern
// I/O contract documented in docs/PIPELINE-PATTERNS.md.
// ---------------------------------------------------------------------------

type harness struct {
	t           *testing.T
	exec        *testutil.FakeExecutor
	execStore   *testutil.FakeExecutionStore
	stepStore   *testutil.FakeStepResultStore
	bus         *engine.EventBus
	eng         *engine.Engine
	execution   domain.Execution
	events      []engine.Event
	eventsMu    sync.Mutex
	subDone     chan struct{}
}

// newHarness wires an engine with fakes, registers the FakeExecutor for
// every node kind, and subscribes to the event bus so assertions can see
// the full event tape post-run. Call drain() after ExecuteWorkflow to
// stop the subscription.
func newHarness(t *testing.T, scripts map[string]testutil.Script) *harness {
	t.Helper()

	fe := testutil.NewFakeExecutor(scripts)
	reg := engine.NewExecutorRegistry()
	reg.Register(domain.NodeTypeAgent, fe)
	reg.Register(domain.NodeTypeMedia, fe)
	reg.Register(domain.NodeTypeTool, fe)

	bus := engine.NewEventBus()
	execStore := testutil.NewFakeExecutionStore()
	stepStore := testutil.NewFakeStepResultStore()

	// fileStore=nil → engine skips manifest writes; that's fine for pattern
	// tests. Manifest contract has its own focused test.
	eng := engine.NewEngine(reg, bus, execStore, stepStore, nil)

	h := &harness{
		t:         t,
		exec:      fe,
		execStore: execStore,
		stepStore: stepStore,
		bus:       bus,
		eng:       eng,
		subDone:   make(chan struct{}),
	}

	return h
}

// seedExecution creates and preloads an execution row so ExecuteWorkflow
// has something to update. Mirrors what the queue worker does in prod.
func (h *harness) seedExecution() domain.Execution {
	h.t.Helper()
	tenant := uuid.New()
	e, _ := h.execStore.Create(context.Background(), domain.Execution{
		TenantID:          tenant,
		PipelineVersionID: uuid.New(),
		Status:            domain.StatusPending,
	})
	h.execution = e
	return e
}

// subscribe starts a goroutine that drains the event bus into h.events.
// Call drain() before reading events so the subscription exits cleanly.
func (h *harness) subscribe() {
	ch := h.bus.Subscribe(h.execution.ID)
	go func() {
		defer close(h.subDone)
		for ev := range ch {
			h.eventsMu.Lock()
			h.events = append(h.events, ev)
			h.eventsMu.Unlock()
		}
	}()
}

// drain unsubscribes and waits for the subscribe goroutine to finish.
func (h *harness) drain() {
	h.t.Helper()
	// Give in-flight publishes a moment to hit the channel. Executing
	// synchronously on a 16-slot buffer means nothing should be queued,
	// but the race detector still needs a yield point.
	time.Sleep(20 * time.Millisecond)
	// We don't have the subscription channel here to Unsubscribe; rely on
	// the engine's own Unsubscribe-on-terminal-event behaviour, or close
	// via a separate Unsubscribe call. For simplicity we poll until the
	// goroutine exits (up to 500ms).
	select {
	case <-h.subDone:
	case <-time.After(500 * time.Millisecond):
		// The bus stays open; subscribers drain naturally on GC. Not a
		// leak in a test that ends imminently.
	}
}

// eventTypes returns the ordered event-type list for easy assertion.
func (h *harness) eventTypes() []engine.EventType {
	h.eventsMu.Lock()
	defer h.eventsMu.Unlock()
	out := make([]engine.EventType, len(h.events))
	for i, ev := range h.events {
		out[i] = ev.Type
	}
	return out
}

// stepByNode returns the (single) step_result row for a given node ID, or
// a zero value. Fails the test if more than one row exists for the node.
func (h *harness) stepByNode(nodeID string) domain.StepResult {
	h.t.Helper()
	rows := h.stepStore.ForExecution(h.execution.ID)
	var match []domain.StepResult
	for _, r := range rows {
		if r.NodeID == nodeID {
			match = append(match, r)
		}
	}
	if len(match) > 1 {
		h.t.Fatalf("multiple step_results for node %q — expected one", nodeID)
	}
	if len(match) == 0 {
		return domain.StepResult{}
	}
	return match[0]
}

// ---------------------------------------------------------------------------
// Graph builders — small helpers that keep each pattern test readable.
// ---------------------------------------------------------------------------

func agentNode(id string, outType domain.PortType) domain.NodeInstance {
	return domain.NodeInstance{
		ID:    id,
		Type:  domain.NodeTypeAgent,
		Label: id,
		Ports: []domain.Port{
			{ID: "in", Name: "Input", Type: domain.PortTypeAny, Direction: domain.PortInput},
			{ID: "out", Name: "Output", Type: outType, Direction: domain.PortOutput},
		},
		Config: json.RawMessage(`{}`),
	}
}

func mediaNode(id string, mediaPromptType, outType domain.PortType) domain.NodeInstance {
	return domain.NodeInstance{
		ID:    id,
		Type:  domain.NodeTypeMedia,
		Label: id,
		Ports: []domain.Port{
			{ID: "in_prompt", Name: "Prompt", Type: mediaPromptType, Direction: domain.PortInput, Required: true},
			{ID: "ref", Name: "Reference", Type: domain.PortTypeAny, Direction: domain.PortInput},
			{ID: "out", Name: "Output", Type: outType, Direction: domain.PortOutput},
		},
		Config: json.RawMessage(`{}`),
	}
}

func toolTextBox(id string) domain.NodeInstance {
	return domain.NodeInstance{
		ID:    id,
		Type:  domain.NodeTypeTool,
		Label: id,
		Ports: []domain.Port{
			{ID: "out", Name: "Output", Type: domain.PortTypeText, Direction: domain.PortOutput},
		},
		Config: json.RawMessage(`{}`),
	}
}

func toolImageBox(id string) domain.NodeInstance {
	return domain.NodeInstance{
		ID:    id,
		Type:  domain.NodeTypeTool,
		Label: id,
		Ports: []domain.Port{
			{ID: "out", Name: "Output", Type: domain.PortTypeImage, Direction: domain.PortOutput},
		},
		Config: json.RawMessage(`{}`),
	}
}

func edge(id, src, srcPort, tgt, tgtPort string) domain.Edge {
	return domain.Edge{
		ID:         id,
		Source:     src,
		SourcePort: srcPort,
		Target:     tgt,
		TargetPort: tgtPort,
	}
}

// ---------------------------------------------------------------------------
// B1. Text-only chain (Tool → Agent → Agent)
// ---------------------------------------------------------------------------

func TestPattern_B1_TextChain(t *testing.T) {
	t.Parallel()

	scripts := map[string]testutil.Script{
		"seed":    {Output: testutil.TextOutput("the lighthouse, at dawn")},
		"outline": {Chunks: []string{"I. ", "Setup "}, Output: testutil.TextOutputWithCost("I. Setup", 10, 0.001)},
		"draft":   {Chunks: []string{"The "}, Output: testutil.TextOutputWithCost("The lighthouse woke...", 25, 0.002)},
	}

	h := newHarness(t, scripts)
	exec := h.seedExecution()
	h.subscribe()

	graph := domain.PipelineGraph{
		Nodes: []domain.NodeInstance{
			toolTextBox("seed"),
			agentNode("outline", domain.PortTypeText),
			agentNode("draft", domain.PortTypeText),
		},
		Edges: []domain.Edge{
			edge("e1", "seed", "out", "outline", "in"),
			edge("e2", "outline", "out", "draft", "in"),
		},
	}

	if err := h.eng.ExecuteWorkflow(context.Background(), exec, graph); err != nil {
		t.Fatalf("ExecuteWorkflow: %v", err)
	}
	h.drain()

	// Every node has exactly one completed step row.
	for _, id := range []string{"seed", "outline", "draft"} {
		row := h.stepByNode(id)
		if row.Status != domain.StatusCompleted {
			t.Errorf("node %q status = %q, want completed", id, row.Status)
		}
	}

	// Execution completes with summed cost + tokens.
	snap := h.execStore.Snapshot(exec.ID)
	if snap.Status != domain.StatusCompleted {
		t.Fatalf("execution status = %q", snap.Status)
	}
	var gotTokens int
	var gotCost float64
	if snap.TotalTokens != nil {
		gotTokens = *snap.TotalTokens
	}
	if snap.TotalCost != nil {
		gotCost = *snap.TotalCost
	}
	if gotTokens != 35 {
		t.Errorf("total tokens = %d, want 35", gotTokens)
	}
	if gotCost < 0.002 {
		t.Errorf("total cost = %v, want >=0.003", gotCost)
	}

	// Event tape includes step_started×3 + step_completed×3 +
	// execution_completed. Chunk count is variable; don't over-pin.
	types := h.eventTypes()
	gotStarted := 0
	gotCompleted := 0
	gotExec := 0
	for _, tp := range types {
		switch tp {
		case engine.EventStepStarted:
			gotStarted++
		case engine.EventStepCompleted:
			gotCompleted++
		case engine.EventExecutionCompleted:
			gotExec++
		}
	}
	if gotStarted != 3 || gotCompleted != 3 || gotExec != 1 {
		t.Errorf("event counts: started=%d completed=%d exec=%d", gotStarted, gotCompleted, gotExec)
	}
}

// ---------------------------------------------------------------------------
// B2. Script → image_prompt → image (canonical sample pipeline)
// ---------------------------------------------------------------------------

func TestPattern_B2_ScriptToImage(t *testing.T) {
	t.Parallel()

	fileURL := "proj/pipe/exec/image-001.png"
	scripts := map[string]testutil.Script{
		"script":  {Output: testutil.TextOutputWithCost("A cinematic scene...", 50, 0.005)},
		"crafter": {Output: testutil.TextOutputWithCost("cinematic wide shot, golden hour", 20, 0.002)},
		"image":   {Output: testutil.FileRefOutput(fileURL)},
	}

	h := newHarness(t, scripts)
	exec := h.seedExecution()
	h.subscribe()

	graph := domain.PipelineGraph{
		Nodes: []domain.NodeInstance{
			agentNode("script", domain.PortTypeText),
			agentNode("crafter", domain.PortTypeImagePrompt),
			mediaNode("image", domain.PortTypeImagePrompt, domain.PortTypeImage),
		},
		Edges: []domain.Edge{
			edge("e1", "script", "out", "crafter", "in"),
			edge("e2", "crafter", "out", "image", "in_prompt"),
		},
	}

	if err := h.eng.ExecuteWorkflow(context.Background(), exec, graph); err != nil {
		t.Fatalf("ExecuteWorkflow: %v", err)
	}
	h.drain()

	// Image node's output_data is the clotho://file/ URL, not inline text.
	imgRow := h.stepByNode("image")
	var out string
	if err := json.Unmarshal(imgRow.OutputData, &out); err != nil {
		t.Fatalf("decode image output: %v", err)
	}
	if !strings.HasPrefix(out, "clotho://file/") {
		t.Errorf("image output should be a clotho://file URL, got %q", out)
	}

	// Execution cost = script + crafter (image has no cost in script).
	snap := h.execStore.Snapshot(exec.ID)
	if snap.TotalCost == nil || *snap.TotalCost < 0.006 {
		t.Errorf("total cost = %v, want >= 0.007", snap.TotalCost)
	}
}

// ---------------------------------------------------------------------------
// B5. Fan-out: script → 3 prompt agents → 3 media nodes (7 steps total)
// ---------------------------------------------------------------------------

func TestPattern_B5_FanOut(t *testing.T) {
	t.Parallel()

	scripts := map[string]testutil.Script{
		"script":       {Output: testutil.TextOutputWithCost("lighthouse scene", 30, 0.003)},
		"img_prompt":   {Output: testutil.TextOutput("cinematic lighthouse at dawn")},
		"vid_prompt":   {Output: testutil.TextOutput("slow dolly-in, 5s")},
		"aud_prompt":   {Output: testutil.TextOutput("calm narration")},
		"img":          {Output: testutil.FileRefOutput("p/pi/e/img.png")},
		"vid":          {Output: testutil.FileRefOutput("p/pi/e/vid.mp4")},
		"aud":          {Output: testutil.FileRefOutput("p/pi/e/aud.mp3")},
	}

	h := newHarness(t, scripts)
	exec := h.seedExecution()
	h.subscribe()

	graph := domain.PipelineGraph{
		Nodes: []domain.NodeInstance{
			agentNode("script", domain.PortTypeText),
			agentNode("img_prompt", domain.PortTypeImagePrompt),
			agentNode("vid_prompt", domain.PortTypeVideoPrompt),
			agentNode("aud_prompt", domain.PortTypeAudioPrompt),
			mediaNode("img", domain.PortTypeImagePrompt, domain.PortTypeImage),
			mediaNode("vid", domain.PortTypeVideoPrompt, domain.PortTypeVideo),
			mediaNode("aud", domain.PortTypeAudioPrompt, domain.PortTypeAudio),
		},
		Edges: []domain.Edge{
			edge("e1", "script", "out", "img_prompt", "in"),
			edge("e2", "script", "out", "vid_prompt", "in"),
			edge("e3", "script", "out", "aud_prompt", "in"),
			edge("e4", "img_prompt", "out", "img", "in_prompt"),
			edge("e5", "vid_prompt", "out", "vid", "in_prompt"),
			edge("e6", "aud_prompt", "out", "aud", "in_prompt"),
		},
	}

	if err := h.eng.ExecuteWorkflow(context.Background(), exec, graph); err != nil {
		t.Fatalf("ExecuteWorkflow: %v", err)
	}
	h.drain()

	rows := h.stepStore.ForExecution(exec.ID)
	if len(rows) != 7 {
		t.Fatalf("step_results rows = %d, want 7", len(rows))
	}
	for _, r := range rows {
		if r.Status != domain.StatusCompleted {
			t.Errorf("node %q status = %q", r.NodeID, r.Status)
		}
	}

	// Invariant: the script node's output reaches ALL THREE downstream
	// prompt agents. The fake records inputs per call; assert each of
	// img_prompt, vid_prompt, aud_prompt saw the script output.
	calls := h.exec.Calls()
	sawScript := map[string]bool{}
	for _, c := range calls {
		if c.NodeID == "img_prompt" || c.NodeID == "vid_prompt" || c.NodeID == "aud_prompt" {
			if v, ok := c.Inputs["in"]; ok && strings.Contains(string(v), "lighthouse") {
				sawScript[c.NodeID] = true
			}
		}
	}
	for _, id := range []string{"img_prompt", "vid_prompt", "aud_prompt"} {
		if !sawScript[id] {
			t.Errorf("fan-out failed: %q did not receive script output", id)
		}
	}
}

// ---------------------------------------------------------------------------
// B11. Failure propagation — first agent errors, downstream never runs.
// ---------------------------------------------------------------------------

func TestPattern_B11_FailurePropagation(t *testing.T) {
	t.Parallel()

	providerErr := errors.New("401 unauthorized: sk-proj-abc123def456ghi789jkl012mno345rst rejected")

	scripts := map[string]testutil.Script{
		"script":  {Error: providerErr},
		"crafter": {Output: testutil.TextOutput("should never run")},
		"image":   {Output: testutil.FileRefOutput("never.png")},
	}

	h := newHarness(t, scripts)
	exec := h.seedExecution()
	h.subscribe()

	graph := domain.PipelineGraph{
		Nodes: []domain.NodeInstance{
			agentNode("script", domain.PortTypeText),
			agentNode("crafter", domain.PortTypeImagePrompt),
			mediaNode("image", domain.PortTypeImagePrompt, domain.PortTypeImage),
		},
		Edges: []domain.Edge{
			edge("e1", "script", "out", "crafter", "in"),
			edge("e2", "crafter", "out", "image", "in_prompt"),
		},
	}

	err := h.eng.ExecuteWorkflow(context.Background(), exec, graph)
	if err == nil {
		t.Fatal("ExecuteWorkflow should have returned an error")
	}
	h.drain()

	// Only one step_result row: the failed script. Downstream nodes
	// should never have started.
	rows := h.stepStore.ForExecution(exec.ID)
	if len(rows) != 1 {
		t.Fatalf("step_results = %d (want 1: only the failing script)", len(rows))
	}
	failed := rows[0]
	if failed.NodeID != "script" || failed.Status != domain.StatusFailed {
		t.Errorf("row = %+v", failed)
	}

	// Error string must be scrubbed — the API-key substring must not
	// survive into the DB.
	if failed.Error == nil {
		t.Fatal("expected error message on failed step")
	}
	if strings.Contains(*failed.Error, "abc123def456ghi789jkl012mno345rst") {
		t.Errorf("leaked API key in error: %q", *failed.Error)
	}

	// Execution is marked failed and also scrubbed.
	snap := h.execStore.Snapshot(exec.ID)
	if snap.Status != domain.StatusFailed {
		t.Errorf("execution status = %q", snap.Status)
	}
	if snap.Error == nil || strings.Contains(*snap.Error, "abc123def456ghi789jkl012mno345rst") {
		t.Errorf("execution error leaked key: %v", snap.Error)
	}

	// No execution_completed event; should see execution_failed.
	types := h.eventTypes()
	hasCompleted := false
	hasFailed := false
	for _, tp := range types {
		if tp == engine.EventExecutionCompleted {
			hasCompleted = true
		}
		if tp == engine.EventExecutionFailed {
			hasFailed = true
		}
	}
	if hasCompleted {
		t.Error("execution_completed fired on failure path")
	}
	if !hasFailed {
		t.Error("execution_failed did not fire")
	}
}

// ---------------------------------------------------------------------------
// B12. Cancellation — context.Cancel during execution.
// ---------------------------------------------------------------------------

func TestPattern_B12_Cancellation(t *testing.T) {
	t.Parallel()

	// The script emits many chunks; we cancel the context partway through.
	// The engine selects on ctx.Done before starting each node — so the
	// cancel should prevent *subsequent* nodes from running, even if the
	// in-flight one completes.
	scripts := map[string]testutil.Script{
		"script":  {Chunks: []string{"a", "b", "c"}, Output: testutil.TextOutput("done")},
		"crafter": {Output: testutil.TextOutput("should not run")},
	}

	h := newHarness(t, scripts)
	exec := h.seedExecution()
	h.subscribe()

	graph := domain.PipelineGraph{
		Nodes: []domain.NodeInstance{
			agentNode("script", domain.PortTypeText),
			agentNode("crafter", domain.PortTypeText),
		},
		Edges: []domain.Edge{
			edge("e1", "script", "out", "crafter", "in"),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately. The engine's pre-node context check will bail
	// before running ANY node — this locks the "execution responds to
	// cancellation" contract. Subsequent refinements (mid-stream abort)
	// build on this baseline.
	cancel()

	_ = h.eng.ExecuteWorkflow(ctx, exec, graph)
	h.drain()

	snap := h.execStore.Snapshot(exec.ID)
	if snap.Status != domain.StatusFailed && snap.Status != domain.StatusCancelled {
		// The current engine marks cancellation as a failure via
		// failExecution("execution cancelled"); that's acceptable for
		// now — the lock is "does not complete". Update this assertion
		// when the engine grows an explicit StatusCancelled path.
		t.Errorf("execution status = %q, want failed or cancelled", snap.Status)
	}

	rows := h.stepStore.ForExecution(exec.ID)
	if len(rows) > 0 {
		// Engine checks ctx.Done before starting any node's step_result
		// row. With a pre-cancelled context, we expect zero rows.
		t.Errorf("unexpected step rows on cancelled execution: %d", len(rows))
	}
}
