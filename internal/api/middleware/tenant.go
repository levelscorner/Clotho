package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const tenantContextKey contextKey = "tenant_id"

// hardcoded Phase 1 tenant
var defaultTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// Tenant injects a hardcoded tenant ID into the request context for Phase 1.
func Tenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), tenantContextKey, defaultTenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TenantIDFromContext extracts the tenant ID from the request context.
func TenantIDFromContext(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(tenantContextKey).(uuid.UUID)
	if !ok {
		return defaultTenantID
	}
	return id
}

// WithTenantIDForTest attaches a tenant ID to the context using the real
// tenant context key. Test-only — production code should go through the
// Auth middleware, which is the only place where trust is established.
func WithTenantIDForTest(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenantID)
}
