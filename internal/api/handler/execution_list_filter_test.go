package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/api/middleware"
	"github.com/user/clotho/internal/domain"
)

func TestExecutionList_StatusFilterReturnsOnlyMatching(t *testing.T) {
	t.Parallel()
	store := newFakeExecutionStore()
	tenant := uuid.New()
	for _, status := range []domain.ExecutionStatus{
		domain.StatusFailed,
		domain.StatusFailed,
		domain.StatusCompleted,
		domain.StatusRunning,
		domain.StatusFailed,
	} {
		store.seed(domain.Execution{TenantID: tenant, Status: status, PipelineVersionID: uuid.New()})
	}

	h := NewExecutionHandler(store, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/executions?status=failed", nil)
	req = req.WithContext(middleware.WithTenantIDForTest(req.Context(), tenant))
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rr.Code, rr.Body.String())
	}
	var got []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("filtered count = %d, want 3 (3 failed in seed)", len(got))
	}
	for _, row := range got {
		if row["status"] != "failed" {
			t.Errorf("status filter leaked: row %v", row)
		}
	}
}

func TestExecutionList_NoFilterReturnsAll(t *testing.T) {
	t.Parallel()
	store := newFakeExecutionStore()
	tenant := uuid.New()
	for i := 0; i < 4; i++ {
		store.seed(domain.Execution{TenantID: tenant, Status: domain.StatusCompleted, PipelineVersionID: uuid.New()})
	}

	h := NewExecutionHandler(store, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/executions", nil)
	req = req.WithContext(middleware.WithTenantIDForTest(req.Context(), tenant))
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var got []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 4 {
		t.Errorf("no filter should return all; got %d, want 4", len(got))
	}
}

// silence unused import lint when only some test files use context.
var _ = context.Background
