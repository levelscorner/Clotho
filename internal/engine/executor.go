package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/user/clotho/internal/domain"
)

// StepOutput wraps the result of a single node execution with optional metadata.
type StepOutput struct {
	Data       json.RawMessage
	TokensUsed *int
	CostUSD    *float64
}

// StepExecutor executes a single node, consuming inputs and producing output.
type StepExecutor interface {
	Execute(ctx context.Context, node domain.NodeInstance, inputs map[string]json.RawMessage) (StepOutput, error)
}

// ExecutorRegistry maps node types to their executors.
type ExecutorRegistry struct {
	executors map[domain.NodeType]StepExecutor
}

// NewExecutorRegistry creates an empty registry.
func NewExecutorRegistry() *ExecutorRegistry {
	return &ExecutorRegistry{
		executors: make(map[domain.NodeType]StepExecutor),
	}
}

// Register adds an executor for a given node type.
func (r *ExecutorRegistry) Register(nodeType domain.NodeType, executor StepExecutor) {
	r.executors[nodeType] = executor
}

// Get returns the executor for a given node type, or an error if not registered.
func (r *ExecutorRegistry) Get(nodeType domain.NodeType) (StepExecutor, error) {
	e, ok := r.executors[nodeType]
	if !ok {
		return nil, fmt.Errorf("no executor registered for node type %q", nodeType)
	}
	return e, nil
}
