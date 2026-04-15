// Package testutil provides in-memory fakes for exercising the engine in
// unit tests without real providers, Postgres, or a filesystem.
//
// The pipeline-pattern suite (internal/engine/pipeline_patterns_test.go)
// uses these fakes to drive the engine end-to-end: graph validation →
// topological order → per-node execution → SSE publication → step_result
// persistence → manifest write. They are scoped to _test builds and never
// ship in the binary.
package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/engine"
)

// Script describes what the FakeExecutor will emit for a given node ID.
// Exactly one of Output / Error / Chunks+Output should be meaningful:
//
//   - Output set, Error nil: non-streaming success.
//   - Error set: returned from Execute / sent on the error channel.
//   - Chunks + Output set: streaming success; chunks emit in order, then
//     Output is posted to the result channel.
//   - Chunks + Error set: streaming failure mid-stream; chunks emit in
//     order, then Error is posted to the error channel.
type Script struct {
	// Output is the final step output (json-encoded bytes + optional
	// tokens/cost). Ignored when Error is set and no chunks are provided.
	Output engine.StepOutput

	// Chunks are emitted on the stream channel in order before Output
	// (or Error) lands on the terminal channel.
	Chunks []string

	// Error, if non-nil, short-circuits the execution: returned from
	// Execute or posted to errCh from ExecuteStream.
	Error error
}

// Call records a single Execute / ExecuteStream invocation. Tests assert
// against the slice to verify input plumbing.
type Call struct {
	NodeID   string
	Stream   bool
	Inputs   map[string]json.RawMessage
	NodeKind domain.NodeType
}

// FakeExecutor is a StepExecutor that returns scripted responses keyed by
// node ID. One fake covers all three node kinds (agent / media / tool) so
// a single instance serves an entire test pipeline.
type FakeExecutor struct {
	mu      sync.Mutex
	scripts map[string]Script // keyed by node.ID
	calls   []Call
}

// NewFakeExecutor creates a fake with the given per-node scripts. Missing
// keys yield an error at execution time ("no script for node X"), so
// forgetting to seed a node is a loud test failure rather than a silent
// pass.
func NewFakeExecutor(scripts map[string]Script) *FakeExecutor {
	if scripts == nil {
		scripts = map[string]Script{}
	}
	return &FakeExecutor{scripts: scripts}
}

// SetScript replaces or adds a script after construction.
func (f *FakeExecutor) SetScript(nodeID string, s Script) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scripts[nodeID] = s
}

// Calls returns a copy of the recorded call log.
func (f *FakeExecutor) Calls() []Call {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Call, len(f.calls))
	copy(out, f.calls)
	return out
}

// Execute satisfies engine.StepExecutor for non-streaming nodes.
func (f *FakeExecutor) Execute(
	_ context.Context,
	node domain.NodeInstance,
	inputs map[string]json.RawMessage,
) (engine.StepOutput, error) {
	f.record(node, inputs, false)
	s, ok := f.scripts[node.ID]
	if !ok {
		return engine.StepOutput{}, fmt.Errorf("fake: no script for node %q", node.ID)
	}
	if s.Error != nil {
		return engine.StepOutput{}, s.Error
	}
	return s.Output, nil
}

// ExecuteStream satisfies engine.StepExecutor for streaming nodes. Emits
// scripted chunks in order then posts either Output or Error to the
// appropriate channel. All channels are closed before this function
// returns — the engine reads on resultCh / errCh so we cannot block.
func (f *FakeExecutor) ExecuteStream(
	_ context.Context,
	node domain.NodeInstance,
	inputs map[string]json.RawMessage,
) (<-chan engine.ExecutorStreamChunk, <-chan engine.StepOutput, <-chan error) {
	f.record(node, inputs, true)

	chunkCh := make(chan engine.ExecutorStreamChunk, 16)
	resultCh := make(chan engine.StepOutput, 1)
	errCh := make(chan error, 1)

	go func() {
		// Only close the chunks channel — the engine drains it via `for
		// chunk := range chunks`. Closing resultCh AND errCh would make
		// the engine's `select { case <-resultCh; case <-errCh }` racy:
		// both closed channels "ready" makes the select pick either one,
		// silently dropping the actual outcome. Instead, send to the
		// appropriate terminal channel and let the other stay open —
		// Go's GC collects them once the goroutine exits.
		defer close(chunkCh)

		s, ok := f.scripts[node.ID]
		if !ok {
			errCh <- fmt.Errorf("fake: no script for node %q", node.ID)
			return
		}
		for _, c := range s.Chunks {
			chunkCh <- engine.ExecutorStreamChunk{Content: c}
		}
		if s.Error != nil {
			errCh <- s.Error
			return
		}
		resultCh <- s.Output
	}()

	return chunkCh, resultCh, errCh
}

func (f *FakeExecutor) record(node domain.NodeInstance, inputs map[string]json.RawMessage, stream bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// Copy inputs so a later test assertion isn't racy with the engine's
	// live map.
	copied := make(map[string]json.RawMessage, len(inputs))
	for k, v := range inputs {
		dup := make(json.RawMessage, len(v))
		copy(dup, v)
		copied[k] = dup
	}
	f.calls = append(f.calls, Call{
		NodeID:   node.ID,
		Stream:   stream,
		Inputs:   copied,
		NodeKind: node.Type,
	})
}

// TextOutput is a tiny helper for the common case of "return a string
// output with no metadata". The engine serializes any json-encodable
// thing, including a plain string.
func TextOutput(s string) engine.StepOutput {
	b, _ := json.Marshal(s)
	return engine.StepOutput{Data: b}
}

// TextOutputWithCost is like TextOutput but attaches token / cost metrics
// so pattern tests can assert roll-ups.
func TextOutputWithCost(s string, tokens int, costUSD float64) engine.StepOutput {
	b, _ := json.Marshal(s)
	return engine.StepOutput{
		Data:       b,
		TokensUsed: &tokens,
		CostUSD:    &costUSD,
	}
}

// FileRefOutput helps media-node scripts: returns a StepOutput whose Data
// is a JSON string of the form "clotho://file/{rel}". The engine normally
// mints this URL inside the media executor; in tests we cut out the disk
// write and just hand the URL back directly.
func FileRefOutput(rel string) engine.StepOutput {
	url := "clotho://file/" + rel
	b, _ := json.Marshal(url)
	return engine.StepOutput{Data: b}
}
