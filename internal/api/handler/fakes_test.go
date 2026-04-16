package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/store"
)

// fakeCredentialStore is an in-memory CredentialStore for handler tests.
// It stores plaintext keys directly (no encryption) so GetDecrypted just
// returns whatever PlaintextKey was seeded.
type fakeCredentialStore struct {
	mu    sync.Mutex
	creds map[uuid.UUID]domain.Credential
	keys  map[uuid.UUID]string
	// failGet, failDecrypt let tests force error paths.
	failGet     bool
	failDecrypt bool
}

func newFakeCredentialStore() *fakeCredentialStore {
	return &fakeCredentialStore{
		creds: make(map[uuid.UUID]domain.Credential),
		keys:  make(map[uuid.UUID]string),
	}
}

func (s *fakeCredentialStore) seed(c domain.Credential) domain.Credential {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	s.creds[c.ID] = c
	s.keys[c.ID] = c.PlaintextKey
	return c
}

func (s *fakeCredentialStore) Create(_ context.Context, c domain.Credential) (domain.Credential, error) {
	return s.seed(c), nil
}

func (s *fakeCredentialStore) Get(_ context.Context, id, tenantID uuid.UUID) (domain.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failGet {
		return domain.Credential{}, fmt.Errorf("forced get failure")
	}
	c, ok := s.creds[id]
	if !ok || (tenantID != uuid.Nil && c.TenantID != tenantID) {
		return domain.Credential{}, fmt.Errorf("not found")
	}
	return c, nil
}

func (s *fakeCredentialStore) GetDecrypted(_ context.Context, id, tenantID uuid.UUID) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failDecrypt {
		return "", fmt.Errorf("forced decrypt failure")
	}
	if _, ok := s.creds[id]; !ok {
		return "", fmt.Errorf("not found")
	}
	_ = tenantID
	return s.keys[id], nil
}

func (s *fakeCredentialStore) ListByTenant(_ context.Context, tenantID uuid.UUID) ([]domain.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.Credential
	for _, c := range s.creds {
		if c.TenantID == tenantID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *fakeCredentialStore) Delete(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.creds, id)
	delete(s.keys, id)
	return nil
}

// Compile-time assertion the fake satisfies the interface.
var _ store.CredentialStore = (*fakeCredentialStore)(nil)

// fakeExecutionStore is an in-memory ExecutionStore for handler tests.
// Mirrors testutil.FakeExecutionStore but lives in the handler package
// to avoid cross-package import cycles in test builds.
type fakeExecutionStore struct {
	mu         sync.Mutex
	executions map[uuid.UUID]domain.Execution
}

func newFakeExecutionStore() *fakeExecutionStore {
	return &fakeExecutionStore{executions: make(map[uuid.UUID]domain.Execution)}
}

func (s *fakeExecutionStore) seed(e domain.Execution) domain.Execution {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	s.executions[e.ID] = e
	return e
}

func (s *fakeExecutionStore) Create(_ context.Context, e domain.Execution) (domain.Execution, error) {
	return s.seed(e), nil
}

func (s *fakeExecutionStore) Get(_ context.Context, id, tenantID uuid.UUID) (domain.Execution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.executions[id]
	if !ok || (tenantID != uuid.Nil && e.TenantID != tenantID) {
		return domain.Execution{}, fmt.Errorf("not found")
	}
	return e, nil
}

func (s *fakeExecutionStore) GetByID(_ context.Context, id uuid.UUID) (domain.Execution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.executions[id]
	if !ok {
		return domain.Execution{}, fmt.Errorf("not found")
	}
	return e, nil
}

func (s *fakeExecutionStore) ListByTenant(_ context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.Execution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []domain.Execution
	for _, e := range s.executions {
		if e.TenantID == tenantID {
			all = append(all, e)
		}
	}
	if offset >= len(all) {
		return nil, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

func (s *fakeExecutionStore) UpdateStatus(_ context.Context, id uuid.UUID, status domain.ExecutionStatus, errMsg *string) error {
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
	s.executions[id] = e
	return nil
}

func (s *fakeExecutionStore) SetFailure(_ context.Context, id uuid.UUID, failureJSON json.RawMessage, errMsg *string) error {
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

func (s *fakeExecutionStore) UpdateCost(_ context.Context, id uuid.UUID, totalCost float64, totalTokens int) error {
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

func (s *fakeExecutionStore) Complete(_ context.Context, id uuid.UUID, totalCost float64, totalTokens int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.executions[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	e.Status = domain.StatusCompleted
	e.TotalCost = &totalCost
	e.TotalTokens = &totalTokens
	s.executions[id] = e
	return nil
}

func (s *fakeExecutionStore) Cancel(_ context.Context, id, _ uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.executions[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	e.Status = domain.StatusCancelled
	s.executions[id] = e
	return nil
}

var _ store.ExecutionStore = (*fakeExecutionStore)(nil)

// fakeJobStore is an in-memory JobStore that records Enqueue calls so
// tests can assert the handler enqueued a retry.
type fakeJobStore struct {
	mu       sync.Mutex
	enqueued []uuid.UUID
}

func newFakeJobStore() *fakeJobStore {
	return &fakeJobStore{}
}

func (s *fakeJobStore) Enqueued() []uuid.UUID {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]uuid.UUID, len(s.enqueued))
	copy(out, s.enqueued)
	return out
}

func (s *fakeJobStore) Enqueue(_ context.Context, executionID uuid.UUID, _ json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enqueued = append(s.enqueued, executionID)
	return nil
}

func (s *fakeJobStore) Dequeue(_ context.Context) (*store.Job, error) {
	return nil, nil
}

func (s *fakeJobStore) Heartbeat(_ context.Context, _ uuid.UUID) error { return nil }
func (s *fakeJobStore) Complete(_ context.Context, _ uuid.UUID) error  { return nil }
func (s *fakeJobStore) Fail(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (s *fakeJobStore) ReapZombies(_ context.Context, _ time.Duration) (int, error) {
	return 0, nil
}

var _ store.JobStore = (*fakeJobStore)(nil)

// fakePipelineStore is defined in tenant_isolation_test.go — reused here.

// fakePipelineVersionStore — minimal subset used by handler tests.
type fakePipelineVersionStore struct {
	mu       sync.Mutex
	versions map[uuid.UUID]domain.PipelineVersion
	byPipe   map[uuid.UUID][]domain.PipelineVersion
}

func newFakePipelineVersionStore() *fakePipelineVersionStore {
	return &fakePipelineVersionStore{
		versions: make(map[uuid.UUID]domain.PipelineVersion),
		byPipe:   make(map[uuid.UUID][]domain.PipelineVersion),
	}
}

func (s *fakePipelineVersionStore) Create(_ context.Context, pv domain.PipelineVersion) (domain.PipelineVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pv.ID == uuid.Nil {
		pv.ID = uuid.New()
	}
	if pv.CreatedAt.IsZero() {
		pv.CreatedAt = time.Now()
	}
	s.versions[pv.ID] = pv
	s.byPipe[pv.PipelineID] = append(s.byPipe[pv.PipelineID], pv)
	return pv, nil
}

func (s *fakePipelineVersionStore) Get(_ context.Context, id uuid.UUID) (domain.PipelineVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pv, ok := s.versions[id]
	if !ok {
		return domain.PipelineVersion{}, fmt.Errorf("not found")
	}
	return pv, nil
}

func (s *fakePipelineVersionStore) GetLatest(_ context.Context, pipelineID uuid.UUID) (domain.PipelineVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.byPipe[pipelineID]
	if len(list) == 0 {
		return domain.PipelineVersion{}, fmt.Errorf("not found")
	}
	best := list[0]
	for _, pv := range list[1:] {
		if pv.Version > best.Version {
			best = pv
		}
	}
	return best, nil
}

func (s *fakePipelineVersionStore) GetByVersion(_ context.Context, pipelineID uuid.UUID, version int) (domain.PipelineVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, pv := range s.byPipe[pipelineID] {
		if pv.Version == version {
			return pv, nil
		}
	}
	return domain.PipelineVersion{}, fmt.Errorf("not found")
}

func (s *fakePipelineVersionStore) ListByPipeline(_ context.Context, pipelineID uuid.UUID) ([]domain.PipelineVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.PipelineVersion, len(s.byPipe[pipelineID]))
	copy(out, s.byPipe[pipelineID])
	return out, nil
}

var _ store.PipelineVersionStore = (*fakePipelineVersionStore)(nil)
