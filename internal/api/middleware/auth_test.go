package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestAuthWithConfig_NoAuthBypass(t *testing.T) {
	var gotUserID uuid.UUID
	var gotTenantID uuid.UUID
	handler := AuthWithConfig(AuthConfig{
		NoAuth:            true,
		AcknowledgeNoAuth: true,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
		gotTenantID = TenantIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/anything", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	want := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	if gotUserID != want {
		t.Errorf("user id = %v, want %v", gotUserID, want)
	}
	if gotTenantID != want {
		t.Errorf("tenant id = %v, want %v", gotTenantID, want)
	}
}

func TestAuthWithConfig_NoAuthRequiresAcknowledge(t *testing.T) {
	// Fail-closed: NoAuth alone must NOT bypass. Missing Authorization should
	// produce 401.
	handler := AuthWithConfig(AuthConfig{
		NoAuth:            true,
		AcknowledgeNoAuth: false,
		JWTSecret:         "test-secret",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/anything", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (bypass must require acknowledge)", rr.Code)
	}
}

func TestAuthWithConfig_JWTPathUnchangedWhenNoAuthFalse(t *testing.T) {
	handler := AuthWithConfig(AuthConfig{
		NoAuth:    false,
		JWTSecret: "test-secret",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// No Authorization header — should be rejected.
	req := httptest.NewRequest(http.MethodGet, "/api/anything", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}

	// Malformed bearer — should be rejected.
	req2 := httptest.NewRequest(http.MethodGet, "/api/anything", nil)
	req2.Header.Set("Authorization", "Bearer not-a-real-token")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for malformed bearer", rr2.Code)
	}
}
