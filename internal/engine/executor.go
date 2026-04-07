package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/domain"
)

type contextKey string

const tenantContextKey contextKey = "engine_tenant_id"

// ContextWithTenantID stores a tenant ID in the context for use by executors.
func ContextWithTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenantID)
}

// TenantIDFromContext retrieves the tenant ID stored in the context by the engine.
func TenantIDFromContext(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(tenantContextKey).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

// StepOutput wraps the result of a single node execution with optional metadata.
type StepOutput struct {
	Data       json.RawMessage
	TokensUsed *int
	CostUSD    *float64
}

// ExecutorStreamChunk is a single piece of streaming output from a node execution.
type ExecutorStreamChunk struct {
	Content string // incremental text content
}

// StepExecutor executes a single node, consuming inputs and producing output.
type StepExecutor interface {
	// Execute runs the node synchronously and returns the complete output.
	Execute(ctx context.Context, node domain.NodeInstance, inputs map[string]json.RawMessage) (StepOutput, error)

	// ExecuteStream runs the node and streams incremental output on the channel.
	// The channel is closed when streaming completes. The final StepOutput is returned.
	// Implementations that don't support streaming should send one chunk and return.
	ExecuteStream(ctx context.Context, node domain.NodeInstance, inputs map[string]json.RawMessage) (<-chan ExecutorStreamChunk, <-chan StepOutput, <-chan error)
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
