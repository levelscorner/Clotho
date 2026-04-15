package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/user/clotho/internal/domain"
)

// ToolExecutor implements StepExecutor for tool nodes (text box, image box, etc.).
type ToolExecutor struct{}

// NewToolExecutor creates a ToolExecutor.
func NewToolExecutor() *ToolExecutor {
	return &ToolExecutor{}
}

// Execute runs a tool node: returns Content or MediaURL as a JSON string.
func (e *ToolExecutor) Execute(_ context.Context, node domain.NodeInstance, _ map[string]json.RawMessage) (StepOutput, error) {
	var cfg domain.ToolNodeConfig
	if err := json.Unmarshal(node.Config, &cfg); err != nil {
		return StepOutput{}, fmt.Errorf("tool executor: unmarshal config: %w", err)
	}

	var output string
	switch {
	case cfg.Content != "":
		output = cfg.Content
	case cfg.MediaURL != "":
		output = cfg.MediaURL
	default:
		output = ""
	}

	data, err := json.Marshal(output)
	if err != nil {
		return StepOutput{}, fmt.Errorf("tool executor: marshal output: %w", err)
	}

	return StepOutput{
		Data:       json.RawMessage(data),
		TokensUsed: nil,
		CostUSD:    nil,
	}, nil
}

// ExecuteStream runs a tool node with a single-chunk stream.
func (e *ToolExecutor) ExecuteStream(ctx context.Context, node domain.NodeInstance, inputs map[string]json.RawMessage) (<-chan ExecutorStreamChunk, <-chan StepOutput, <-chan error) {
	chunks := make(chan ExecutorStreamChunk, 1)
	result := make(chan StepOutput, 1)
	errCh := make(chan error, 1)

	go func() {
		// Only close chunks (the engine iterates it with `for range`).
		// Closing result AND errCh together causes the engine's
		// `select { case <-result; case <-errCh }` to race — see the
		// comment in agent_executor.go for the full failure mode. Send
		// to exactly one of result / errCh and let them GC on exit.
		defer close(chunks)

		out, err := e.Execute(ctx, node, inputs)
		if err != nil {
			errCh <- err
			return
		}

		// Send the complete output as a single chunk
		var content string
		_ = json.Unmarshal(out.Data, &content)
		chunks <- ExecutorStreamChunk{Content: content}
		result <- out
	}()

	return chunks, result, errCh
}
