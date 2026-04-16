package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/user/clotho/internal/api/dto"
	"github.com/user/clotho/internal/api/middleware"
	"github.com/user/clotho/internal/domain"
)

// helper: build a request whose context has a tenant ID, route via chi
// so chi.URLParam(r, "id") resolves correctly.
func buildTestReq(t *testing.T, method, path, urlPattern string, urlParam string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if urlParam != "" {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", urlParam)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}
	tenant := uuid.New()
	req = req.WithContext(middleware.WithTenantIDForTest(req.Context(), tenant))
	return req
}

func TestCredentialTest_BadUUIDReturns400(t *testing.T) {
	t.Parallel()
	store := newFakeCredentialStore()
	h := NewCredentialHandler(store)

	req := buildTestReq(t, http.MethodPost, "/api/credentials/not-a-uuid/test", "", "not-a-uuid")
	rr := httptest.NewRecorder()
	h.Test(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (body=%s)", rr.Code, rr.Body.String())
	}
}

func TestCredentialTest_NotFoundReturns404(t *testing.T) {
	t.Parallel()
	store := newFakeCredentialStore()
	h := NewCredentialHandler(store)

	missing := uuid.NewString()
	req := buildTestReq(t, http.MethodPost, "/api/credentials/"+missing+"/test", "", missing)
	rr := httptest.NewRecorder()
	h.Test(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestCredentialTest_DecryptFailureReturns500(t *testing.T) {
	t.Parallel()
	store := newFakeCredentialStore()
	cred := store.seed(domain.Credential{
		Provider:     "openai",
		Label:        "test",
		PlaintextKey: "sk-x",
	})
	store.failDecrypt = true
	h := NewCredentialHandler(store)

	req := buildTestReq(t, http.MethodPost, "/api/credentials/"+cred.ID.String()+"/test", "", cred.ID.String())
	req = req.WithContext(middleware.WithTenantIDForTest(req.Context(), cred.TenantID))
	rr := httptest.NewRecorder()
	h.Test(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rr.Code)
	}
}

func TestCredentialTest_UnsupportedProviderReturnsOkFalse(t *testing.T) {
	t.Parallel()
	// Ollama isn't in createProviderFromCredential — credential test
	// should report ok=false at the HTTP-200 layer rather than 500.
	store := newFakeCredentialStore()
	cred := store.seed(domain.Credential{
		Provider:     "ollama",
		Label:        "local",
		PlaintextKey: "anything",
	})
	h := NewCredentialHandler(store)

	req := buildTestReq(t, http.MethodPost, "/api/credentials/"+cred.ID.String()+"/test", "", cred.ID.String())
	req = req.WithContext(middleware.WithTenantIDForTest(req.Context(), cred.TenantID))
	rr := httptest.NewRecorder()
	h.Test(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rr.Code, rr.Body.String())
	}
	var body dto.CredentialTestResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.OK {
		t.Errorf("ok = true for unsupported provider; want false")
	}
	if body.Provider != "ollama" {
		t.Errorf("provider = %q, want ollama", body.Provider)
	}
}
