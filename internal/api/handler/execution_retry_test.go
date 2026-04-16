package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/user/clotho/internal/api/middleware"
	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/queue"
)

func newRetryHandler(execStore *fakeExecutionStore, jobs *fakeJobStore) *ExecutionHandler {
	q := queue.NewQueue(jobs)
	// Pipelines/versions/steps not needed for Retry; pass nil to keep
	// the test focused on the retry path.
	return NewExecutionHandler(execStore, nil, nil, nil, q)
}

func reqWithRouteAndTenant(t *testing.T, method, path, id string, tenant uuid.UUID) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(middleware.WithTenantIDForTest(req.Context(), tenant))
	return req
}

func TestExecutionRetry_HappyPath(t *testing.T) {
	t.Parallel()
	execStore := newFakeExecutionStore()
	jobs := newFakeJobStore()
	tenant := uuid.New()
	pvID := uuid.New()
	original := execStore.seed(domain.Execution{
		PipelineVersionID: pvID,
		TenantID:          tenant,
		Status:            domain.StatusFailed,
	})

	h := newRetryHandler(execStore, jobs)
	req := reqWithRouteAndTenant(t, http.MethodPost,
		"/api/executions/"+original.ID.String()+"/retry",
		original.ID.String(), tenant)
	rr := httptest.NewRecorder()
	h.Retry(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%s)", rr.Code, rr.Body.String())
	}

	// Response should carry the NEW execution ID (not the original).
	var got struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID == original.ID.String() {
		t.Errorf("retry returned original ID; want a fresh one")
	}

	// Job should have been enqueued for the NEW execution.
	enq := jobs.Enqueued()
	if len(enq) != 1 {
		t.Fatalf("enqueued = %d jobs, want 1", len(enq))
	}
	if enq[0].String() != got.ID {
		t.Errorf("enqueued ID = %s, want new exec ID %s", enq[0], got.ID)
	}

	// New execution must point at the SAME pipeline_version_id.
	clone, err := execStore.GetByID(context.Background(), enq[0])
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if clone.PipelineVersionID != pvID {
		t.Errorf("retry pinned to wrong version: got %s want %s", clone.PipelineVersionID, pvID)
	}
}

func TestExecutionRetry_BadUUID400(t *testing.T) {
	t.Parallel()
	h := newRetryHandler(newFakeExecutionStore(), newFakeJobStore())
	req := reqWithRouteAndTenant(t, http.MethodPost, "/api/executions/bad/retry", "bad", uuid.New())
	rr := httptest.NewRecorder()
	h.Retry(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestExecutionRetry_NotFound404(t *testing.T) {
	t.Parallel()
	h := newRetryHandler(newFakeExecutionStore(), newFakeJobStore())
	missing := uuid.NewString()
	req := reqWithRouteAndTenant(t, http.MethodPost, "/api/executions/"+missing+"/retry", missing, uuid.New())
	rr := httptest.NewRecorder()
	h.Retry(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestExecutionRetry_TenantIsolation(t *testing.T) {
	t.Parallel()
	execStore := newFakeExecutionStore()
	tenantA := uuid.New()
	tenantB := uuid.New()
	original := execStore.seed(domain.Execution{
		PipelineVersionID: uuid.New(),
		TenantID:          tenantA,
		Status:            domain.StatusFailed,
	})

	h := newRetryHandler(execStore, newFakeJobStore())
	// Tenant B tries to retry tenant A's execution → 404.
	req := reqWithRouteAndTenant(t, http.MethodPost,
		"/api/executions/"+original.ID.String()+"/retry",
		original.ID.String(), tenantB)
	rr := httptest.NewRecorder()
	h.Retry(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("cross-tenant retry status = %d, want 404 (no enumeration leak)", rr.Code)
	}
}
