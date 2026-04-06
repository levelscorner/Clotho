package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/store"
)

// Queue wraps a JobStore with convenience methods for enqueuing executions.
type Queue struct {
	jobs store.JobStore
}

// NewQueue creates a new Queue.
func NewQueue(jobs store.JobStore) *Queue {
	return &Queue{jobs: jobs}
}

// Submit enqueues an execution for background processing.
func (q *Queue) Submit(ctx context.Context, executionID uuid.UUID, payload json.RawMessage) error {
	if err := q.jobs.Enqueue(ctx, executionID, payload); err != nil {
		return fmt.Errorf("queue submit: %w", err)
	}
	return nil
}

// Jobs returns the underlying JobStore for direct access by the worker.
func (q *Queue) Jobs() store.JobStore {
	return q.jobs
}
