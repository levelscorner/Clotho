package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/clotho/internal/store"
)

// JobStore implements store.JobStore using Postgres SKIP LOCKED.
type JobStore struct {
	pool     *pgxpool.Pool
	workerID string
}

func NewJobStore(pool *pgxpool.Pool, workerID string) *JobStore {
	return &JobStore{pool: pool, workerID: workerID}
}

func (s *JobStore) Enqueue(ctx context.Context, executionID uuid.UUID, payload json.RawMessage) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO job_queue (execution_id, payload)
		 VALUES ($1, $2)`,
		executionID, payload,
	)
	if err != nil {
		return fmt.Errorf("job enqueue: %w", err)
	}
	return nil
}

func (s *JobStore) Dequeue(ctx context.Context) (*store.Job, error) {
	var j store.Job
	var payloadJSON []byte
	err := s.pool.QueryRow(ctx,
		`UPDATE job_queue
		 SET status = 'running', claimed_by = $1, claimed_at = now(), last_ping = now()
		 WHERE id = (
		     SELECT id FROM job_queue
		     WHERE status = 'pending'
		     ORDER BY created_at
		     FOR UPDATE SKIP LOCKED
		     LIMIT 1
		 )
		 RETURNING id, execution_id, status, payload, claimed_by, claimed_at, last_ping, created_at`,
		s.workerID,
	).Scan(
		&j.ID, &j.ExecutionID, &j.Status, &payloadJSON,
		&j.ClaimedBy, &j.ClaimedAt, &j.LastPing, &j.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("job dequeue: %w", err)
	}
	if payloadJSON != nil {
		j.Payload = json.RawMessage(payloadJSON)
	}
	return &j, nil
}

func (s *JobStore) Heartbeat(ctx context.Context, jobID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE job_queue SET last_ping = now() WHERE id = $1 AND status = 'running'`,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("job heartbeat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("job heartbeat: not found or not running")
	}
	return nil
}

func (s *JobStore) Complete(ctx context.Context, jobID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE job_queue SET status = 'completed' WHERE id = $1`,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("job complete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("job complete: not found")
	}
	return nil
}

func (s *JobStore) Fail(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE job_queue SET status = 'failed', payload = jsonb_set(COALESCE(payload, '{}'), '{error}', to_jsonb($2::text)) WHERE id = $1`,
		jobID, errMsg,
	)
	if err != nil {
		return fmt.Errorf("job fail: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("job fail: not found")
	}
	return nil
}

func (s *JobStore) ReapZombies(ctx context.Context, timeout time.Duration) (int, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE job_queue
		 SET status = 'pending', claimed_by = NULL, claimed_at = NULL, last_ping = NULL
		 WHERE status = 'running' AND last_ping < now() - make_interval(secs => $1)`,
		timeout.Seconds(),
	)
	if err != nil {
		return 0, fmt.Errorf("job reap zombies: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
