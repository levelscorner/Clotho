package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/domain"
)

// Job represents a row in the job_queue table.
type Job struct {
	ID          uuid.UUID       `json:"id"`
	ExecutionID uuid.UUID       `json:"execution_id"`
	Status      string          `json:"status"`
	Payload     json.RawMessage `json:"payload"`
	ClaimedBy   *string         `json:"claimed_by,omitempty"`
	ClaimedAt   *time.Time      `json:"claimed_at,omitempty"`
	LastPing    *time.Time      `json:"last_ping,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// ProjectStore manages project CRUD.
type ProjectStore interface {
	Create(ctx context.Context, p domain.Project) (domain.Project, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Project, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.Project, error)
	Update(ctx context.Context, p domain.Project) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// PipelineStore manages pipeline CRUD.
type PipelineStore interface {
	Create(ctx context.Context, p domain.Pipeline) (domain.Pipeline, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Pipeline, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.Pipeline, error)
	Update(ctx context.Context, p domain.Pipeline) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// PipelineVersionStore manages immutable pipeline version snapshots.
type PipelineVersionStore interface {
	Create(ctx context.Context, pv domain.PipelineVersion) (domain.PipelineVersion, error)
	Get(ctx context.Context, id uuid.UUID) (domain.PipelineVersion, error)
	GetLatest(ctx context.Context, pipelineID uuid.UUID) (domain.PipelineVersion, error)
	GetByVersion(ctx context.Context, pipelineID uuid.UUID, version int) (domain.PipelineVersion, error)
	ListByPipeline(ctx context.Context, pipelineID uuid.UUID) ([]domain.PipelineVersion, error)
}

// ExecutionStore manages execution lifecycle.
type ExecutionStore interface {
	Create(ctx context.Context, e domain.Execution) (domain.Execution, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Execution, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.Execution, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ExecutionStatus, errMsg *string) error
	UpdateCost(ctx context.Context, id uuid.UUID, totalCost float64, totalTokens int) error
	Complete(ctx context.Context, id uuid.UUID, totalCost float64, totalTokens int) error
}

// StepResultStore manages per-node step results within an execution.
type StepResultStore interface {
	Create(ctx context.Context, sr domain.StepResult) (domain.StepResult, error)
	ListByExecution(ctx context.Context, executionID uuid.UUID) ([]domain.StepResult, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ExecutionStatus, outputData json.RawMessage, errMsg *string, tokensUsed *int, costUSD *float64, durationMs *int64) error
}

// PresetStore manages agent presets (built-in + tenant-specific).
type PresetStore interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.AgentPreset, error)
	Get(ctx context.Context, id uuid.UUID) (domain.AgentPreset, error)
	Create(ctx context.Context, p domain.AgentPreset) (domain.AgentPreset, error)
	Update(ctx context.Context, p domain.AgentPreset) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// CredentialStore manages LLM provider credentials.
type CredentialStore interface {
	Create(ctx context.Context, c domain.Credential) (domain.Credential, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Credential, error)
	GetDecrypted(ctx context.Context, id uuid.UUID) (string, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.Credential, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// UserStore manages user CRUD.
type UserStore interface {
	Create(ctx context.Context, u domain.User) (domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

// RefreshTokenStore manages refresh token persistence.
type RefreshTokenStore interface {
	Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	Validate(ctx context.Context, userID uuid.UUID, tokenHash string) (bool, error)
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}

// JobStore manages the Postgres-backed job queue using SKIP LOCKED.
type JobStore interface {
	Enqueue(ctx context.Context, executionID uuid.UUID, payload json.RawMessage) error
	Dequeue(ctx context.Context) (*Job, error)
	Heartbeat(ctx context.Context, jobID uuid.UUID) error
	Complete(ctx context.Context, jobID uuid.UUID) error
	Fail(ctx context.Context, jobID uuid.UUID, errMsg string) error
	ReapZombies(ctx context.Context, timeout time.Duration) (int, error)
}
