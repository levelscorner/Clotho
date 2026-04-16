package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/domain"
)

// FakeExecutionStore is an in-memory implementation of store.ExecutionStore
// scoped to tests. Only the methods the engine actually calls during
// ExecuteWorkflow / RerunFromNode are meaningful; the rest are stubs that
// return empty slices.
type FakeExecutionStore struct {
	mu         sync.Mutex
	executions map[uuid.UUID]domain.Execution
}

func NewFakeExecutionStore() *FakeExecutionStore {
	return &FakeExecutionStore{executions: map[uuid.UUID]domain.Execution{}}
}

// Seed preloads an execution; used by tests that call ExecuteWorkflow
// directly (the engine expects the execution row to already exist).
func (s *FakeExecutionStore) Seed(e domain.Execution) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executions[e.ID] = e
}

// Snapshot returns the current state of a single execution, or a zero value
// if the ID doesn't match. Lets tests assert final status/cost/tokens.
func (s *FakeExecutionStore) Snapshot(id uuid.UUID) domain.Execution {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.executions[id]
}

func (s *FakeExecutionStore) Create(_ context.Context, e domain.Execution) (domain.Execution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	e.CreatedAt = time.Now()
	s.executions[e.ID] = e
	return e, nil
}

func (s *FakeExecutionStore) Get(_ context.Context, id, tenantID uuid.UUID) (domain.Execution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.executions[id]
	if !ok || e.TenantID != tenantID {
		return domain.Execution{}, fmt.Errorf("not found")
	}
	return e, nil
}

func (s *FakeExecutionStore) GetByID(_ context.Context, id uuid.UUID) (domain.Execution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.executions[id]
	if !ok {
		return domain.Execution{}, fmt.Errorf("not found")
	}
	return e, nil
}

func (s *FakeExecutionStore) ListByTenant(_ context.Context, _ uuid.UUID, _, _ int) ([]domain.Execution, error) {
	return nil, nil
}

func (s *FakeExecutionStore) UpdateStatus(_ context.Context, id uuid.UUID, status domain.ExecutionStatus, errMsg *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.executions[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	e.Status = status
	if errMsg != nil {
		e.Error = errMsg
	}
	now := time.Now()
	switch status {
	case domain.StatusRunning:
		e.StartedAt = &now
	case domain.StatusCompleted, domain.StatusFailed, domain.StatusCancelled:
		e.CompletedAt = &now
	}
	s.executions[id] = e
	return nil
}

func (s *FakeExecutionStore) SetFailure(_ context.Context, id uuid.UUID, failureJSON json.RawMessage, errMsg *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.executions[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	e.FailureJSON = failureJSON
	if errMsg != nil {
		e.Error = errMsg
	}
	s.executions[id] = e
	return nil
}

func (s *FakeExecutionStore) UpdateCost(_ context.Context, id uuid.UUID, totalCost float64, totalTokens int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.executions[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	e.TotalCost = &totalCost
	e.TotalTokens = &totalTokens
	s.executions[id] = e
	return nil
}

func (s *FakeExecutionStore) Complete(_ context.Context, id uuid.UUID, totalCost float64, totalTokens int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.executions[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	e.Status = domain.StatusCompleted
	e.TotalCost = &totalCost
	e.TotalTokens = &totalTokens
	now := time.Now()
	e.CompletedAt = &now
	s.executions[id] = e
	return nil
}

func (s *FakeExecutionStore) Cancel(_ context.Context, id, tenantID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.executions[id]
	if !ok || e.TenantID != tenantID {
		return fmt.Errorf("not found")
	}
	if e.Status != domain.StatusPending && e.Status != domain.StatusRunning {
		return fmt.Errorf("not cancellable")
	}
	e.Status = domain.StatusCancelled
	now := time.Now()
	e.CompletedAt = &now
	s.executions[id] = e
	return nil
}

// FakeStepResultStore is an in-memory step_results table. Records are kept
// in the order they were Created so tests can assert the execution order.
type FakeStepResultStore struct {
	mu      sync.Mutex
	results []domain.StepResult
	byID    map[uuid.UUID]int // index into results
}

func NewFakeStepResultStore() *FakeStepResultStore {
	return &FakeStepResultStore{byID: map[uuid.UUID]int{}}
}

// All returns a snapshot of every step result row, in creation order.
func (s *FakeStepResultStore) All() []domain.StepResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.StepResult, len(s.results))
	copy(out, s.results)
	return out
}

// ForExecution returns rows for one execution in creation order.
func (s *FakeStepResultStore) ForExecution(execID uuid.UUID) []domain.StepResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.StepResult
	for _, r := range s.results {
		if r.ExecutionID == execID {
			out = append(out, r)
		}
	}
	return out
}

func (s *FakeStepResultStore) Create(_ context.Context, sr domain.StepResult) (domain.StepResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sr.ID == uuid.Nil {
		sr.ID = uuid.New()
	}
	now := time.Now()
	sr.StartedAt = &now
	s.byID[sr.ID] = len(s.results)
	s.results = append(s.results, sr)
	return sr, nil
}

func (s *FakeStepResultStore) ListByExecution(_ context.Context, executionID uuid.UUID) ([]domain.StepResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.StepResult
	for _, r := range s.results {
		if r.ExecutionID == executionID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (s *FakeStepResultStore) UpdateStatus(
	_ context.Context,
	id uuid.UUID,
	status domain.ExecutionStatus,
	outputData json.RawMessage,
	errMsg *string,
	tokensUsed *int,
	costUSD *float64,
	durationMs *int64,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, ok := s.byID[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	r := s.results[idx]
	r.Status = status
	r.OutputData = outputData
	r.Error = errMsg
	r.TokensUsed = tokensUsed
	r.CostUSD = costUSD
	r.DurationMs = durationMs
	now := time.Now()
	r.CompletedAt = &now
	s.results[idx] = r
	return nil
}

func (s *FakeStepResultStore) SetFailure(_ context.Context, id uuid.UUID, failureJSON json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, ok := s.byID[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	r := s.results[idx]
	r.FailureJSON = failureJSON
	s.results[idx] = r
	return nil
}

// FakePipelineVersionStore is a minimal in-memory PipelineVersion store.
// The engine itself only reads, never writes, so Create is provided for
// test setup and the read methods return whatever Seed() was called with.
type FakePipelineVersionStore struct {
	mu       sync.Mutex
	versions map[uuid.UUID]domain.PipelineVersion
	byPipe   map[uuid.UUID][]domain.PipelineVersion
}

func NewFakePipelineVersionStore() *FakePipelineVersionStore {
	return &FakePipelineVersionStore{
		versions: map[uuid.UUID]domain.PipelineVersion{},
		byPipe:   map[uuid.UUID][]domain.PipelineVersion{},
	}
}

// Seed stores a version so Get / GetLatest / GetByVersion can find it.
func (s *FakePipelineVersionStore) Seed(pv domain.PipelineVersion) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.versions[pv.ID] = pv
	s.byPipe[pv.PipelineID] = append(s.byPipe[pv.PipelineID], pv)
}

func (s *FakePipelineVersionStore) Create(_ context.Context, pv domain.PipelineVersion) (domain.PipelineVersion, error) {
	s.Seed(pv)
	return pv, nil
}

func (s *FakePipelineVersionStore) Get(_ context.Context, id uuid.UUID) (domain.PipelineVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pv, ok := s.versions[id]
	if !ok {
		return domain.PipelineVersion{}, fmt.Errorf("not found")
	}
	return pv, nil
}

func (s *FakePipelineVersionStore) GetLatest(_ context.Context, pipelineID uuid.UUID) (domain.PipelineVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.byPipe[pipelineID]
	if len(list) == 0 {
		return domain.PipelineVersion{}, fmt.Errorf("not found")
	}
	// Highest version number wins.
	best := list[0]
	for _, pv := range list[1:] {
		if pv.Version > best.Version {
			best = pv
		}
	}
	return best, nil
}

func (s *FakePipelineVersionStore) GetByVersion(_ context.Context, pipelineID uuid.UUID, version int) (domain.PipelineVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, pv := range s.byPipe[pipelineID] {
		if pv.Version == version {
			return pv, nil
		}
	}
	return domain.PipelineVersion{}, fmt.Errorf("not found")
}

func (s *FakePipelineVersionStore) ListByPipeline(_ context.Context, pipelineID uuid.UUID) ([]domain.PipelineVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.PipelineVersion, len(s.byPipe[pipelineID]))
	copy(out, s.byPipe[pipelineID])
	return out, nil
}
