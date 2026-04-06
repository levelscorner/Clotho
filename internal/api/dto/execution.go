package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/domain"
)

// ExecutePipelineRequest is the request body for executing a pipeline (empty for now).
type ExecutePipelineRequest struct{}

// ExecutionResponse is the API response for an execution.
type ExecutionResponse struct {
	ID                uuid.UUID              `json:"id"`
	PipelineVersionID uuid.UUID              `json:"pipeline_version_id"`
	TenantID          uuid.UUID              `json:"tenant_id"`
	Status            domain.ExecutionStatus `json:"status"`
	TotalCost         *float64               `json:"total_cost,omitempty"`
	TotalTokens       *int                   `json:"total_tokens,omitempty"`
	Error             *string                `json:"error,omitempty"`
	StartedAt         *time.Time             `json:"started_at,omitempty"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	Steps             []StepResultResponse   `json:"steps,omitempty"`
}

// StepResultResponse is the API response for a step result.
type StepResultResponse struct {
	ID          uuid.UUID              `json:"id"`
	ExecutionID uuid.UUID              `json:"execution_id"`
	NodeID      string                 `json:"node_id"`
	Status      domain.ExecutionStatus `json:"status"`
	InputData   json.RawMessage        `json:"input_data,omitempty"`
	OutputData  json.RawMessage        `json:"output_data,omitempty"`
	Error       *string                `json:"error,omitempty"`
	TokensUsed  *int                   `json:"tokens_used,omitempty"`
	CostUSD     *float64               `json:"cost_usd,omitempty"`
	DurationMs  *int64                 `json:"duration_ms,omitempty"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// ExecutionFromDomain converts a domain.Execution to ExecutionResponse.
func ExecutionFromDomain(e domain.Execution) ExecutionResponse {
	return ExecutionResponse{
		ID:                e.ID,
		PipelineVersionID: e.PipelineVersionID,
		TenantID:          e.TenantID,
		Status:            e.Status,
		TotalCost:         e.TotalCost,
		TotalTokens:       e.TotalTokens,
		Error:             e.Error,
		StartedAt:         e.StartedAt,
		CompletedAt:       e.CompletedAt,
		CreatedAt:         e.CreatedAt,
	}
}

// ExecutionWithSteps converts an execution and its step results to ExecutionResponse.
func ExecutionWithSteps(e domain.Execution, steps []domain.StepResult) ExecutionResponse {
	resp := ExecutionFromDomain(e)
	resp.Steps = StepResultsFromDomain(steps)
	return resp
}

// StepResultFromDomain converts a domain.StepResult to StepResultResponse.
func StepResultFromDomain(sr domain.StepResult) StepResultResponse {
	return StepResultResponse{
		ID:          sr.ID,
		ExecutionID: sr.ExecutionID,
		NodeID:      sr.NodeID,
		Status:      sr.Status,
		InputData:   sr.InputData,
		OutputData:  sr.OutputData,
		Error:       sr.Error,
		TokensUsed:  sr.TokensUsed,
		CostUSD:     sr.CostUSD,
		DurationMs:  sr.DurationMs,
		StartedAt:   sr.StartedAt,
		CompletedAt: sr.CompletedAt,
	}
}

// StepResultsFromDomain converts a slice of domain.StepResult to StepResultResponse slice.
func StepResultsFromDomain(steps []domain.StepResult) []StepResultResponse {
	out := make([]StepResultResponse, 0, len(steps))
	for _, sr := range steps {
		out = append(out, StepResultFromDomain(sr))
	}
	return out
}

// ExecutionsFromDomain converts a slice of domain.Execution to ExecutionResponse slice.
func ExecutionsFromDomain(execs []domain.Execution) []ExecutionResponse {
	out := make([]ExecutionResponse, 0, len(execs))
	for _, e := range execs {
		out = append(out, ExecutionFromDomain(e))
	}
	return out
}
