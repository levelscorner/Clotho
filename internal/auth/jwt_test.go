package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/domain"
)

func testUser() domain.User {
	return domain.User{
		ID:       uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		TenantID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Email:    "test@example.com",
		Name:     "Test User",
		IsActive: true,
	}
}

const testSecret = "super-secret-signing-key-for-tests"

func TestGenerateAndValidateToken(t *testing.T) {
	user := testUser()

	token, err := GenerateAccessToken(user, testSecret, 5*time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := ValidateToken(token, testSecret)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("UserID mismatch: got %s, want %s", claims.UserID, user.ID)
	}
	if claims.TenantID != user.TenantID {
		t.Errorf("TenantID mismatch: got %s, want %s", claims.TenantID, user.TenantID)
	}
	if claims.Email != user.Email {
		t.Errorf("Email mismatch: got %s, want %s", claims.Email, user.Email)
	}
	if claims.Subject != user.ID.String() {
		t.Errorf("Subject mismatch: got %s, want %s", claims.Subject, user.ID.String())
	}
}

func TestValidateExpiredToken(t *testing.T) {
	user := testUser()

	// Generate with 1ms expiry
	token, err := GenerateAccessToken(user, testSecret, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	// Wait for expiry
	time.Sleep(10 * time.Millisecond)

	_, err = ValidateToken(token, testSecret)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestValidateInvalidSignature(t *testing.T) {
	user := testUser()

	token, err := GenerateAccessToken(user, "secret-A", 5*time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	_, err = ValidateToken(token, "secret-B")
	if err == nil {
		t.Fatal("expected error for invalid signature, got nil")
	}
}

func TestRefreshTokenUniqueness(t *testing.T) {
	token1, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("first GenerateRefreshToken: %v", err)
	}

	token2, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("second GenerateRefreshToken: %v", err)
	}

	if token1 == "" || token2 == "" {
		t.Fatal("refresh tokens must not be empty")
	}

	if token1 == token2 {
		t.Errorf("refresh tokens should be unique, got same value: %s", token1)
	}
}

func TestHashRefreshToken(t *testing.T) {
	token := "some-refresh-token-value"

	hash1 := HashRefreshToken(token)
	hash2 := HashRefreshToken(token)

	if hash1 == "" {
		t.Fatal("hash must not be empty")
	}

	if hash1 != hash2 {
		t.Errorf("hash should be deterministic: got %s and %s", hash1, hash2)
	}

	// Different input should produce different hash
	other := HashRefreshToken("different-token")
	if hash1 == other {
		t.Error("different tokens should produce different hashes")
	}
}
