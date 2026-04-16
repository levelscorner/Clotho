package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ExecutionStatus tracks the lifecycle of an execution or step.
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
	StatusSkipped   ExecutionStatus = "skipped"
)

// Execution represents a single run of a pipeline version.
//
// FailureJSON carries the structured StepFailure for the FIRST failed step
// (the one that caused the execution to fail). Error stays as a 1-line
// human summary for back-compat with anything still reading it; the rich
// payload lives in FailureJSON. A successful execution leaves both nil.
type Execution struct {
	ID                uuid.UUID       `json:"id"`
	PipelineVersionID uuid.UUID       `json:"pipeline_version_id"`
	TenantID          uuid.UUID       `json:"tenant_id"`
	Status            ExecutionStatus `json:"status"`
	TotalCost         *float64        `json:"total_cost,omitempty"`
	TotalTokens       *int            `json:"total_tokens,omitempty"`
	Error             *string         `json:"error,omitempty"`
	FailureJSON       json.RawMessage `json:"failure,omitempty"`
	TraceID           *string         `json:"trace_id,omitempty"` // OTel root span ID for diagnostics
	StartedAt         *time.Time      `json:"started_at,omitempty"`
	CompletedAt       *time.Time      `json:"completed_at,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
}

// StepResult captures the input/output of a single node execution.
//
// FailureJSON carries the StepFailure for this step when Status == failed.
// Error stays for back-compat (1-line summary); FailureJSON has the
// retryability flag, hint, attempts count, and stage info the UI needs.
type StepResult struct {
	ID          uuid.UUID       `json:"id"`
	ExecutionID uuid.UUID       `json:"execution_id"`
	NodeID      string          `json:"node_id"`
	Status      ExecutionStatus `json:"status"`
	InputData   json.RawMessage `json:"input_data,omitempty"`
	OutputData  json.RawMessage `json:"output_data,omitempty"`
	Error       *string         `json:"error,omitempty"`
	FailureJSON json.RawMessage `json:"failure,omitempty"`
	TokensUsed  *int            `json:"tokens_used,omitempty"`
	CostUSD     *float64        `json:"cost_usd,omitempty"`
	DurationMs  *int64          `json:"duration_ms,omitempty"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
}
