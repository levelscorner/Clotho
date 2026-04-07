package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/clotho/internal/domain"
)

// ExecutionStore implements store.ExecutionStore.
type ExecutionStore struct {
	pool *pgxpool.Pool
}

func NewExecutionStore(pool *pgxpool.Pool) *ExecutionStore {
	return &ExecutionStore{pool: pool}
}

func (s *ExecutionStore) Create(ctx context.Context, e domain.Execution) (domain.Execution, error) {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO executions (id, pipeline_version_id, tenant_id, status)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at`,
		e.ID, e.PipelineVersionID, e.TenantID, e.Status,
	).Scan(&e.CreatedAt)
	if err != nil {
		return domain.Execution{}, fmt.Errorf("execution create: %w", err)
	}
	return e, nil
}

func (s *ExecutionStore) Get(ctx context.Context, id, tenantID uuid.UUID) (domain.Execution, error) {
	var e domain.Execution
	err := s.pool.QueryRow(ctx,
		`SELECT id, pipeline_version_id, tenant_id, status,
		        total_cost, total_tokens, error,
		        started_at, completed_at, created_at
		 FROM executions WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	).Scan(
		&e.ID, &e.PipelineVersionID, &e.TenantID, &e.Status,
		&e.TotalCost, &e.TotalTokens, &e.Error,
		&e.StartedAt, &e.CompletedAt, &e.CreatedAt,
	)
	if err != nil {
		return domain.Execution{}, fmt.Errorf("execution get: %w", err)
	}
	return e, nil
}

// GetByID retrieves an execution by ID without tenant scoping (for internal system use).
func (s *ExecutionStore) GetByID(ctx context.Context, id uuid.UUID) (domain.Execution, error) {
	var e domain.Execution
	err := s.pool.QueryRow(ctx,
		`SELECT id, pipeline_version_id, tenant_id, status,
		        total_cost, total_tokens, error,
		        started_at, completed_at, created_at
		 FROM executions WHERE id = $1`, id,
	).Scan(
		&e.ID, &e.PipelineVersionID, &e.TenantID, &e.Status,
		&e.TotalCost, &e.TotalTokens, &e.Error,
		&e.StartedAt, &e.CompletedAt, &e.CreatedAt,
	)
	if err != nil {
		return domain.Execution{}, fmt.Errorf("execution get by id: %w", err)
	}
	return e, nil
}

func (s *ExecutionStore) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.Execution, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, pipeline_version_id, tenant_id, status,
		        total_cost, total_tokens, error,
		        started_at, completed_at, created_at
		 FROM executions
		 WHERE tenant_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`, tenantID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("execution list: %w", err)
	}
	defer rows.Close()

	var executions []domain.Execution
	for rows.Next() {
		var e domain.Execution
		if err := rows.Scan(
			&e.ID, &e.PipelineVersionID, &e.TenantID, &e.Status,
			&e.TotalCost, &e.TotalTokens, &e.Error,
			&e.StartedAt, &e.CompletedAt, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("execution list scan: %w", err)
		}
		executions = append(executions, e)
	}
	return executions, rows.Err()
}

func (s *ExecutionStore) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ExecutionStatus, errMsg *string) error {
	var query string
	var args []any

	switch status {
	case domain.StatusRunning:
		query = `UPDATE executions SET status = $1, started_at = now() WHERE id = $2`
		args = []any{status, id}
	case domain.StatusCompleted:
		query = `UPDATE executions SET status = $1, completed_at = now() WHERE id = $2`
		args = []any{status, id}
	case domain.StatusFailed:
		query = `UPDATE executions SET status = $1, error = $2, completed_at = now() WHERE id = $3`
		args = []any{status, errMsg, id}
	default:
		query = `UPDATE executions SET status = $1 WHERE id = $2`
		args = []any{status, id}
	}

	tag, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("execution update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("execution update status: not found")
	}
	return nil
}

func (s *ExecutionStore) UpdateCost(ctx context.Context, id uuid.UUID, totalCost float64, totalTokens int) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE executions SET total_cost = $1, total_tokens = $2 WHERE id = $3`,
		totalCost, totalTokens, id,
	)
	if err != nil {
		return fmt.Errorf("execution update cost: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("execution update cost: not found")
	}
	return nil
}

// Complete atomically sets status=completed, cost, tokens, and completed_at in a single UPDATE.
func (s *ExecutionStore) Complete(ctx context.Context, id uuid.UUID, totalCost float64, totalTokens int) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE executions SET status = 'completed', total_cost = $1, total_tokens = $2, completed_at = now() WHERE id = $3`,
		totalCost, totalTokens, id,
	)
	if err != nil {
		return fmt.Errorf("execution complete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("execution complete: not found")
	}
	return nil
}

func (s *ExecutionStore) Cancel(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE executions SET status = $1, completed_at = NOW()
		 WHERE id = $2 AND tenant_id = $3 AND status IN ('pending', 'running')`,
		domain.StatusCancelled, id, tenantID)
	if err != nil {
		return fmt.Errorf("execution cancel: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("execution cancel: not found or not cancellable")
	}
	return nil
}

// StepResultStore implements store.StepResultStore.
type StepResultStore struct {
	pool *pgxpool.Pool
}

func NewStepResultStore(pool *pgxpool.Pool) *StepResultStore {
	return &StepResultStore{pool: pool}
}

func (s *StepResultStore) Create(ctx context.Context, sr domain.StepResult) (domain.StepResult, error) {
	if sr.ID == uuid.Nil {
		sr.ID = uuid.New()
	}

	var inputJSON, outputJSON []byte
	if sr.InputData != nil {
		inputJSON = sr.InputData
	}
	if sr.OutputData != nil {
		outputJSON = sr.OutputData
	}

	err := s.pool.QueryRow(ctx,
		`INSERT INTO step_results (id, execution_id, node_id, status, input_data, output_data)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING started_at`,
		sr.ID, sr.ExecutionID, sr.NodeID, sr.Status, inputJSON, outputJSON,
	).Scan(&sr.StartedAt)
	if err != nil {
		return domain.StepResult{}, fmt.Errorf("step result create: %w", err)
	}
	return sr, nil
}

func (s *StepResultStore) ListByExecution(ctx context.Context, executionID uuid.UUID) ([]domain.StepResult, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, execution_id, node_id, status,
		        input_data, output_data, error,
		        tokens_used, cost_usd, duration_ms,
		        started_at, completed_at
		 FROM step_results
		 WHERE execution_id = $1
		 ORDER BY started_at ASC NULLS LAST`, executionID,
	)
	if err != nil {
		return nil, fmt.Errorf("step result list: %w", err)
	}
	defer rows.Close()

	var results []domain.StepResult
	for rows.Next() {
		var sr domain.StepResult
		var inputJSON, outputJSON []byte
		if err := rows.Scan(
			&sr.ID, &sr.ExecutionID, &sr.NodeID, &sr.Status,
			&inputJSON, &outputJSON, &sr.Error,
			&sr.TokensUsed, &sr.CostUSD, &sr.DurationMs,
			&sr.StartedAt, &sr.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("step result list scan: %w", err)
		}
		if inputJSON != nil {
			sr.InputData = json.RawMessage(inputJSON)
		}
		if outputJSON != nil {
			sr.OutputData = json.RawMessage(outputJSON)
		}
		results = append(results, sr)
	}
	return results, rows.Err()
}

func (s *StepResultStore) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status domain.ExecutionStatus,
	outputData json.RawMessage,
	errMsg *string,
	tokensUsed *int,
	costUSD *float64,
	durationMs *int64,
) error {
	var outputJSON []byte
	if outputData != nil {
		outputJSON = outputData
	}

	tag, err := s.pool.Exec(ctx,
		`UPDATE step_results
		 SET status = $1, output_data = $2, error = $3,
		     tokens_used = $4, cost_usd = $5, duration_ms = $6,
		     completed_at = now()
		 WHERE id = $7`,
		status, outputJSON, errMsg,
		tokensUsed, costUSD, durationMs,
		id,
	)
	if err != nil {
		return fmt.Errorf("step result update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("step result update status: not found")
	}
	return nil
}
