package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/user/clotho/internal/api/middleware"
	"github.com/user/clotho/internal/domain"
)

// fakePipelineStore is a stand-in PipelineStore for handler tests. It records
// the (id, tenantID) pair passed to Get so callers can assert the handler
// plumbs the tenant from context into the store call.
type fakePipelineStore struct {
	pipelinesByOwner map[uuid.UUID]map[uuid.UUID]domain.Pipeline // tenantID -> id -> pipeline
	lastTenantQuery  uuid.UUID
}

func newFakePipelineStore() *fakePipelineStore {
	return &fakePipelineStore{
		pipelinesByOwner: make(map[uuid.UUID]map[uuid.UUID]domain.Pipeline),
	}
}

func (f *fakePipelineStore) seed(tenantID uuid.UUID, p domain.Pipeline) {
	if _, ok := f.pipelinesByOwner[tenantID]; !ok {
		f.pipelinesByOwner[tenantID] = make(map[uuid.UUID]domain.Pipeline)
	}
	f.pipelinesByOwner[tenantID][p.ID] = p
}

func (f *fakePipelineStore) Create(_ context.Context, p domain.Pipeline) (domain.Pipeline, error) {
	return p, nil
}

func (f *fakePipelineStore) Get(_ context.Context, id, tenantID uuid.UUID) (domain.Pipeline, error) {
	f.lastTenantQuery = tenantID
	byID, ok := f.pipelinesByOwner[tenantID]
	if !ok {
		return domain.Pipeline{}, errors.New("pipeline not found")
	}
	p, ok := byID[id]
	if !ok {
		return domain.Pipeline{}, errors.New("pipeline not found")
	}
	return p, nil
}

func (f *fakePipelineStore) GetByID(_ context.Context, id uuid.UUID) (domain.Pipeline, error) {
	for _, byID := range f.pipelinesByOwner {
		if p, ok := byID[id]; ok {
			return p, nil
		}
	}
	return domain.Pipeline{}, errors.New("pipeline not found")
}

func (f *fakePipelineStore) ListByProject(_ context.Context, _ uuid.UUID) ([]domain.Pipeline, error) {
	return nil, nil
}

func (f *fakePipelineStore) Update(_ context.Context, _ domain.Pipeline, _ uuid.UUID) error {
	return nil
}

func (f *fakePipelineStore) Delete(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

// fakeProjectStore is a stand-in ProjectStore for handler tests.
type fakeProjectStore struct {
	projects map[uuid.UUID]domain.Project // id -> project (tenant lives on the row)
}

func newFakeProjectStore() *fakeProjectStore {
	return &fakeProjectStore{projects: make(map[uuid.UUID]domain.Project)}
}

func (f *fakeProjectStore) Create(_ context.Context, p domain.Project) (domain.Project, error) {
	return p, nil
}

func (f *fakeProjectStore) Get(_ context.Context, id, tenantID uuid.UUID) (domain.Project, error) {
	p, ok := f.projects[id]
	if !ok || p.TenantID != tenantID {
		return domain.Project{}, errors.New("project not found")
	}
	return p, nil
}

func (f *fakeProjectStore) GetByID(_ context.Context, id uuid.UUID) (domain.Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return domain.Project{}, errors.New("project not found")
	}
	return p, nil
}

func (f *fakeProjectStore) List(_ context.Context, _ uuid.UUID) ([]domain.Project, error) {
	return nil, nil
}

func (f *fakeProjectStore) Update(_ context.Context, _ domain.Project, _ uuid.UUID) error {
	return nil
}

func (f *fakeProjectStore) Delete(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

// withTenant returns a request whose context carries the given tenant ID —
// simulates the auth middleware without wiring up JWT.
func withTenant(r *http.Request, tenantID uuid.UUID) *http.Request {
	return r.WithContext(middleware.WithTenantIDForTest(r.Context(), tenantID))
}

func TestPipelineGet_CrossTenantReturns404(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	pipelineID := uuid.New()

	pipelines := newFakePipelineStore()
	pipelines.seed(tenantA, domain.Pipeline{ID: pipelineID, Name: "A's pipeline"})

	projects := newFakeProjectStore()
	h := NewPipelineHandler(pipelines, projects, nil)

	r := chi.NewRouter()
	r.Route("/api/pipelines", func(r chi.Router) {
		r.Get("/{id}", h.Get)
	})

	// Tenant B requests tenant A's pipeline — must get 404, and must have
	// queried with tenant B (not tenant A) so we know the scope was applied.
	req := httptest.NewRequest(http.MethodGet, "/api/pipelines/"+pipelineID.String(), nil)
	req = withTenant(req, tenantB)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-tenant read, got %d", rec.Code)
	}
	if pipelines.lastTenantQuery != tenantB {
		t.Fatalf("expected store.Get called with tenantB (%s), got %s", tenantB, pipelines.lastTenantQuery)
	}
}

func TestPipelineGet_SameTenantReturns200(t *testing.T) {
	tenantA := uuid.New()
	pipelineID := uuid.New()

	pipelines := newFakePipelineStore()
	pipelines.seed(tenantA, domain.Pipeline{ID: pipelineID, Name: "A's pipeline"})

	projects := newFakeProjectStore()
	h := NewPipelineHandler(pipelines, projects, nil)

	r := chi.NewRouter()
	r.Route("/api/pipelines", func(r chi.Router) {
		r.Get("/{id}", h.Get)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/pipelines/"+pipelineID.String(), nil)
	req = withTenant(req, tenantA)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for same-tenant read, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["id"] != pipelineID.String() {
		t.Fatalf("expected id %s in body, got %v", pipelineID, body["id"])
	}
}

func TestProjectGet_CrossTenantReturns404(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	projectID := uuid.New()

	projects := newFakeProjectStore()
	projects.projects[projectID] = domain.Project{ID: projectID, TenantID: tenantA, Name: "A's project"}

	h := NewProjectHandler(projects)

	r := chi.NewRouter()
	r.Route("/api/projects", func(r chi.Router) {
		r.Get("/{id}", h.Get)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+projectID.String(), nil)
	req = withTenant(req, tenantB)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-tenant project read, got %d", rec.Code)
	}
}
