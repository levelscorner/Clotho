package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/engine/testutil"
)

// ---------------------------------------------------------------------------
// B15. Pinned node short-circuits the executor
//
// Pinning a node freezes its output. The engine must (a) skip the executor
// entirely (b) propagate PinnedOutput to downstream consumers (c) still
// emit step_completed so the canvas reflects the no-op step. This is the
// $$-saving "iterate the leaf without re-paying the upstream LLM" workflow.
// ---------------------------------------------------------------------------

func TestPattern_B15_PinnedSkipsExecutor(t *testing.T) {
	t.Parallel()

	// Script the upstream agent to FAIL. If the engine actually calls the
	// executor, the test breaks. The pin must skip the call entirely.
	scripts := map[string]testutil.Script{
		"upstream":   {Error: errors.New("upstream should never be called when pinned")},
		"downstream": {Output: testutil.TextOutputWithCost("greeting Hi from pinned", 5, 0.0001)},
	}

	h := newHarness(t, scripts)
	exec := h.seedExecution()
	h.subscribe()

	pinned := json.RawMessage(`"frozen value"`)
	graph := domain.PipelineGraph{
		Nodes: []domain.NodeInstance{
			{
				ID:           "upstream",
				Type:         domain.NodeTypeAgent,
				Label:        "upstream",
				Pinned:       true,
				PinnedOutput: pinned,
				Ports: []domain.Port{
					{ID: "out", Direction: domain.PortOutput, Type: domain.PortTypeText},
				},
				Config: json.RawMessage(`{}`),
			},
			agentNode("downstream", domain.PortTypeText),
		},
		Edges: []domain.Edge{
			edge("e1", "upstream", "out", "downstream", "in"),
		},
	}

	if err := h.eng.ExecuteWorkflow(context.Background(), exec, graph); err != nil {
		t.Fatalf("ExecuteWorkflow: %v (pinned upstream should not have been called)", err)
	}
	h.drain()

	// Upstream should NOT appear in the executor's call log; downstream
	// should have been invoked once with the pinned value as input.
	calls := h.exec.Calls()
	for _, c := range calls {
		if c.NodeID == "upstream" {
			t.Errorf("pinned node was executed: %+v", c)
		}
	}

	downstream := h.stepByNode("downstream")
	if downstream.Status != domain.StatusCompleted {
		t.Errorf("downstream status = %q, want completed", downstream.Status)
	}

	upstream := h.stepByNode("upstream")
	if upstream.Status != domain.StatusCompleted {
		t.Errorf("pinned upstream should still record a completed step, got %q", upstream.Status)
	}
	if string(upstream.OutputData) != string(pinned) {
		t.Errorf("upstream output = %q, want %q", string(upstream.OutputData), string(pinned))
	}
}

// ---------------------------------------------------------------------------
// B16. on_failure=skip keeps downstream running
//
// A node configured with OnFailureSkip records the failure but the engine
// keeps marching. Downstream nodes that did NOT depend on the failed node
// run normally; nodes that did depend get an empty input map for the
// missing port. The execution finishes with status=completed (the run as
// a whole succeeded; one branch was skipped).
// ---------------------------------------------------------------------------

func TestPattern_B16_OnFailureSkipContinuesPipeline(t *testing.T) {
	t.Parallel()

	scripts := map[string]testutil.Script{
		"flaky":       {Error: errors.New("provider returned 401")},
		"independent": {Output: testutil.TextOutput("ran fine on the side")},
	}

	h := newHarness(t, scripts)
	exec := h.seedExecution()
	h.subscribe()

	graph := domain.PipelineGraph{
		Nodes: []domain.NodeInstance{
			{
				ID:        "flaky",
				Type:      domain.NodeTypeAgent,
				Label:     "flaky",
				OnFailure: domain.OnFailureSkip,
				Ports: []domain.Port{
					{ID: "out", Direction: domain.PortOutput, Type: domain.PortTypeText},
				},
				Config: json.RawMessage(`{}`),
			},
			agentNode("independent", domain.PortTypeText),
		},
		// No edges — both nodes are independent so the failed one
		// shouldn't block the other.
	}

	if err := h.eng.ExecuteWorkflow(context.Background(), exec, graph); err != nil {
		t.Fatalf("ExecuteWorkflow returned error despite OnFailureSkip: %v", err)
	}
	h.drain()

	flaky := h.stepByNode("flaky")
	if flaky.Status != domain.StatusFailed {
		t.Errorf("flaky step status = %q, want failed (skip records failure)", flaky.Status)
	}
	independent := h.stepByNode("independent")
	if independent.Status != domain.StatusCompleted {
		t.Errorf("independent step status = %q, want completed (skip should not abort)", independent.Status)
	}

	snap := h.execStore.Snapshot(exec.ID)
	if snap.Status != domain.StatusCompleted {
		t.Errorf("execution status = %q, want completed (skip pipeline finishes)", snap.Status)
	}
}
