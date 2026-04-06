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
