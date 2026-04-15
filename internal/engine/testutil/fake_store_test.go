package testutil

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/domain"
)

func TestFakeExecutionStore_CreateAndGet(t *testing.T) {
	t.Parallel()

	s := NewFakeExecutionStore()
	tenantA := uuid.New()
	tenantB := uuid.New()

	created, err := s.Create(context.Background(), domain.Execution{TenantID: tenantA, Status: domain.StatusPending})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == uuid.Nil {
		t.Fatal("Create should assign an ID when one is not supplied")
	}

	got, err := s.Get(context.Background(), created.ID, tenantA)
	if err != nil {
		t.Fatalf("get own-tenant: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("round-trip ID mismatch")
	}

	// Cross-tenant read must miss — mirrors the real store's tenant scope.
	if _, err := s.Get(context.Background(), created.ID, tenantB); err == nil {
		t.Fatal("cross-tenant read should fail")
	}
}

func TestFakeExecutionStore_UpdateStatusAndComplete(t *testing.T) {
	t.Parallel()

	s := NewFakeExecutionStore()
	created, _ := s.Create(context.Background(), domain.Execution{TenantID: uuid.New(), Status: domain.StatusPending})

	if err := s.UpdateStatus(context.Background(), created.ID, domain.StatusRunning, nil); err != nil {
		t.Fatalf("running: %v", err)
	}
	if snap := s.Snapshot(created.ID); snap.Status != domain.StatusRunning || snap.StartedAt == nil {
		t.Fatalf("snapshot after running = %+v", snap)
	}

	if err := s.Complete(context.Background(), created.ID, 1.23, 456); err != nil {
		t.Fatalf("complete: %v", err)
	}
	snap := s.Snapshot(created.ID)
	if snap.Status != domain.StatusCompleted {
		t.Fatalf("status = %q", snap.Status)
	}
	if snap.TotalCost == nil || *snap.TotalCost != 1.23 {
		t.Fatalf("total cost = %v", snap.TotalCost)
	}
	if snap.TotalTokens == nil || *snap.TotalTokens != 456 {
		t.Fatalf("total tokens = %v", snap.TotalTokens)
	}
}

func TestFakeStepResultStore_CreateListUpdate(t *testing.T) {
	t.Parallel()

	s := NewFakeStepResultStore()
	execID := uuid.New()

	a, _ := s.Create(context.Background(), domain.StepResult{ExecutionID: execID, NodeID: "a", Status: domain.StatusRunning})
	b, _ := s.Create(context.Background(), domain.StepResult{ExecutionID: execID, NodeID: "b", Status: domain.StatusRunning})

	list, _ := s.ListByExecution(context.Background(), execID)
	if len(list) != 2 {
		t.Fatalf("list len = %d", len(list))
	}
	if list[0].NodeID != "a" || list[1].NodeID != "b" {
		t.Fatalf("order = %q,%q", list[0].NodeID, list[1].NodeID)
	}

	tokens := 10
	cost := 0.05
	duration := int64(123)
	if err := s.UpdateStatus(context.Background(), b.ID, domain.StatusCompleted, nil, nil, &tokens, &cost, &duration); err != nil {
		t.Fatalf("update: %v", err)
	}

	list, _ = s.ListByExecution(context.Background(), execID)
	if list[1].Status != domain.StatusCompleted {
		t.Fatalf("status = %q", list[1].Status)
	}
	if list[1].TokensUsed == nil || *list[1].TokensUsed != 10 {
		t.Fatalf("tokens = %v", list[1].TokensUsed)
	}
	_ = a // keep reference to silence unused
}

func TestFakePipelineVersionStore_GetLatestByVersion(t *testing.T) {
	t.Parallel()

	s := NewFakePipelineVersionStore()
	pipeID := uuid.New()
	s.Seed(domain.PipelineVersion{ID: uuid.New(), PipelineID: pipeID, Version: 1})
	s.Seed(domain.PipelineVersion{ID: uuid.New(), PipelineID: pipeID, Version: 3})
	s.Seed(domain.PipelineVersion{ID: uuid.New(), PipelineID: pipeID, Version: 2})

	latest, err := s.GetLatest(context.Background(), pipeID)
	if err != nil || latest.Version != 3 {
		t.Fatalf("latest = %+v, err=%v", latest, err)
	}
	v2, err := s.GetByVersion(context.Background(), pipeID, 2)
	if err != nil || v2.Version != 2 {
		t.Fatalf("v2 = %+v, err=%v", v2, err)
	}
	if _, err := s.GetByVersion(context.Background(), pipeID, 99); err == nil {
		t.Fatal("missing version should error")
	}
}
