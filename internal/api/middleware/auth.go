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

// Auth returns middleware that validates JWT bearer tokens and injects
// UserID and TenantID into the request context.
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				writeUnauthorized(w)
				return
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := auth.ValidateToken(tokenStr, jwtSecret)
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
