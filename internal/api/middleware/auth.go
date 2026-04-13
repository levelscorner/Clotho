package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/auth"
)

const (
	userContextKey   contextKey = "user_id"
	authTenantCtxKey contextKey = "auth_tenant_id"
)

// Local-dev identity used when auth is bypassed via NO_AUTH=true.
// Matches the row seeded in migrations/006_seed_local_dev_user.up.sql.
var (
	localDevUserID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	localDevTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	localDevEmail    = "you@local"
)

const localDevEmailContextKey contextKey = "user_email"

// AuthConfig bundles options for the Auth middleware. Keeps the call site
// backward-compatible via the Auth() helper and lets NO_AUTH be wired in
// without a breaking signature change.
type AuthConfig struct {
	JWTSecret         string
	NoAuth            bool
	AcknowledgeNoAuth bool
}

// Auth returns middleware that validates JWT bearer tokens and injects
// UserID and TenantID into the request context.
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return AuthWithConfig(AuthConfig{JWTSecret: jwtSecret})
}

// AuthWithConfig returns middleware that either validates JWT bearer tokens
// or, when cfg.NoAuth && cfg.AcknowledgeNoAuth, bypasses auth entirely and
// injects a local-dev identity. The bypass path is intentionally fail-closed:
// both flags must be true.
func AuthWithConfig(cfg AuthConfig) func(http.Handler) http.Handler {
	bypass := cfg.NoAuth && cfg.AcknowledgeNoAuth

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if bypass {
				ctx := r.Context()
				ctx = context.WithValue(ctx, userContextKey, localDevUserID)
				ctx = context.WithValue(ctx, authTenantCtxKey, localDevTenantID)
				ctx = context.WithValue(ctx, tenantContextKey, localDevTenantID)
				ctx = context.WithValue(ctx, localDevEmailContextKey, localDevEmail)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				writeUnauthorized(w)
				return
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := auth.ValidateToken(tokenStr, cfg.JWTSecret)
			if err != nil {
				writeUnauthorized(w)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, userContextKey, claims.UserID)
			ctx = context.WithValue(ctx, authTenantCtxKey, claims.TenantID)
			// Also set the legacy tenant context key for backward compat
			ctx = context.WithValue(ctx, tenantContextKey, claims.TenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the authenticated user ID from the context.
func UserIDFromContext(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(userContextKey).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
