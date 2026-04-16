package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/user/clotho/internal/api/middleware"
	"github.com/user/clotho/internal/domain"
)

// helper that builds a SaveVersion request and routes it through chi.
func saveVersionReq(t *testing.T, pipelineID uuid.UUID, tenant uuid.UUID, graph any) *http.Request {
	t.Helper()
	body, err := json.Marshal(map[string]any{"graph": graph})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost,
		"/api/pipelines/"+pipelineID.String()+"/versions", bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", pipelineID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(middleware.WithTenantIDForTest(req.Context(), tenant))
	return req
}

func newSaveHandler(t *testing.T) (*PipelineHandler, uuid.UUID, uuid.UUID) {
	t.Helper()
	pipes := newFakePipelineStore()
	tenant := uuid.New()
	pid := uuid.New()
	pipes.seed(tenant, domain.Pipeline{ID: pid, Name: "p"})
	versions := newFakePipelineVersionStore()
	// projects fake not used by SaveVersion path.
	return NewPipelineHandler(pipes, nil, versions), pid, tenant
}

func TestSaveVersion_EmptyGraphPasses(t *testing.T) {
	t.Parallel()
	h, pid, tenant := newSaveHandler(t)

	req := saveVersionReq(t, pid, tenant, map[string]any{
		"nodes": []any{},
		"edges": []any{},
	})
	rr := httptest.NewRecorder()
	h.SaveVersion(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("empty graph status = %d, want 201 (body=%s)", rr.Code, rr.Body.String())
	}
}

func TestSaveVersion_DanglingEdgeReturns400WithValidationErrors(t *testing.T) {
	t.Parallel()
	h, pid, tenant := newSaveHandler(t)

	// Edge references nodes that don't exist → validation fails.
	graph := map[string]any{
		"nodes": []any{},
		"edges": []any{
			map[string]any{
				"id":          "e1",
				"source":      "ghost",
				"source_port": "out",
				"target":      "phantom",
				"target_port": "in",
			},
		},
	}
	req := saveVersionReq(t, pid, tenant, graph)
	rr := httptest.NewRecorder()
	h.SaveVersion(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body=%s)", rr.Code, rr.Body.String())
	}
	var body struct {
		Error            string                 `json:"error"`
		ValidationErrors []map[string]string    `json:"validation_errors"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v (raw=%s)", err, rr.Body.String())
	}
	if body.Error == "" {
		t.Errorf("expected error string in body")
	}
	if len(body.ValidationErrors) == 0 {
		t.Errorf("expected validation_errors array; got empty")
	}
	// Field path should reference the offending edge.
	foundEdgeRef := false
	for _, ve := range body.ValidationErrors {
		if ve["field"] != "" && contains(ve["field"], "edges[e1]") {
			foundEdgeRef = true
			break
		}
	}
	if !foundEdgeRef {
		t.Errorf("expected at least one validation error referencing edges[e1]; got %v", body.ValidationErrors)
	}
}

func TestSaveVersion_NilNodesArrayReturns400(t *testing.T) {
	t.Parallel()
	h, pid, tenant := newSaveHandler(t)

	req := saveVersionReq(t, pid, tenant, map[string]any{}) // no nodes key
	rr := httptest.NewRecorder()
	h.SaveVersion(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for missing nodes array", rr.Code)
	}
}

// helper: avoid pulling in strings package for one Contains call.
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
