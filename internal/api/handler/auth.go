package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/user/clotho/internal/api/dto"
	"github.com/user/clotho/internal/auth"
	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/store"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	users         store.UserStore
	refreshTokens store.RefreshTokenStore
	jwtSecret     string
	jwtExpiry     time.Duration
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(users store.UserStore, refreshTokens store.RefreshTokenStore, jwtSecret string, jwtExpiry time.Duration) *AuthHandler {
	return &AuthHandler{
		users:         users,
		refreshTokens: refreshTokens,
		jwtSecret:     jwtSecret,
		jwtExpiry:     jwtExpiry,
	}
}

// Routes registers auth routes on the given router (public, no auth middleware).
func (h *AuthHandler) Routes(r chi.Router) {
	r.Post("/api/auth/register", h.Register)
	r.Post("/api/auth/login", h.Login)
	r.Post("/api/auth/refresh", h.Refresh)
}

// Register handles POST /api/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Check email uniqueness
	if _, err := h.users.GetByEmail(r.Context(), req.Email); err == nil {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}

	// Hash password
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		slog.Error("hash password failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	// Create tenant for the user
	tenantID := uuid.New()

	user, err := h.users.Create(r.Context(), domain.User{
		TenantID:     tenantID,
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: hash,
		IsActive:     true,
	})
	if err != nil {
		slog.Error("create user failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	resp, err := h.generateAuthResponse(r.Context(), user)
	if err != nil {
		slog.Error("generate tokens failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if !user.IsActive {
		writeError(w, http.StatusUnauthorized, "account is deactivated")
		return
	}

	if err := auth.ComparePassword(user.PasswordHash, req.Password); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if err := h.users.UpdateLastLogin(r.Context(), user.ID); err != nil {
		slog.Error("update last login failed", "error", err)
	}

	resp, err := h.generateAuthResponse(r.Context(), user)
	if err != nil {
		slog.Error("generate tokens failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to login")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Refresh handles POST /api/auth/refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	// Extract user_id from the Authorization header (the old access token, possibly expired)
	// For simplicity, require user_id in the refresh token request or decode from old JWT.
	// We'll require the caller to still pass a (possibly expired) access token for user identification.
	header := r.Header.Get("Authorization")
	if header == "" || !strings.HasPrefix(header, "Bearer ") {
		writeError(w, http.StatusUnauthorized, "authorization header required for refresh")
		return
	}

	tokenStr := strings.TrimPrefix(header, "Bearer ")
	// Parse without validation (token may be expired)
	claims, err := auth.ValidateToken(tokenStr, h.jwtSecret)
	if err != nil {
		// Try to get user ID from expired token by parsing without time validation
		// For simplicity, we require a valid (non-expired) token or user_id in body.
		writeError(w, http.StatusUnauthorized, "invalid access token")
		return
	}

	tokenHash := auth.HashRefreshToken(req.RefreshToken)
	valid, err := h.refreshTokens.Validate(r.Context(), claims.UserID, tokenHash)
	if err != nil || !valid {
		writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	// Delete old refresh tokens for this user
	if err := h.refreshTokens.DeleteByUser(r.Context(), claims.UserID); err != nil {
		slog.Error("delete old refresh tokens failed", "error", err)
	}

	user, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	resp, err := h.generateAuthResponse(r.Context(), user)
	if err != nil {
		slog.Error("generate tokens failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to refresh tokens")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) generateAuthResponse(ctx context.Context, user domain.User) (dto.AuthResponse, error) {
	accessToken, err := auth.GenerateAccessToken(user, h.jwtSecret, h.jwtExpiry)
	if err != nil {
		return dto.AuthResponse{}, err
	}

	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return dto.AuthResponse{}, err
	}

	refreshHash := auth.HashRefreshToken(refreshToken)
	refreshExpiry := time.Now().Add(7 * 24 * time.Hour) // 7 days
	if err := h.refreshTokens.Create(ctx, user.ID, refreshHash, refreshExpiry); err != nil {
		return dto.AuthResponse{}, err
	}

	return dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: dto.UserResponse{
			ID:       user.ID,
			TenantID: user.TenantID,
			Email:    user.Email,
			Name:     user.Name,
		},
	}, nil
}
