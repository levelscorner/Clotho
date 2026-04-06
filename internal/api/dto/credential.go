package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/domain"
)

// CreateCredentialRequest is the request body for creating a credential.
type CreateCredentialRequest struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Label    string `json:"label"`
}

// CredentialResponse is the API response for a credential (API key masked).
type CredentialResponse struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Provider  string    `json:"provider"`
	APIKey    string    `json:"api_key"` // masked
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
}

// CredentialFromDomain converts a domain.Credential to CredentialResponse with a masked key.
func CredentialFromDomain(c domain.Credential) CredentialResponse {
	return CredentialResponse{
		ID:        c.ID,
		TenantID:  c.TenantID,
		Provider:  c.Provider,
		APIKey:    maskAPIKey(c.APIKey),
		Label:     c.Label,
		CreatedAt: c.CreatedAt,
	}
}

// CredentialsFromDomain converts a slice of domain.Credential to CredentialResponse slice.
func CredentialsFromDomain(creds []domain.Credential) []CredentialResponse {
	out := make([]CredentialResponse, 0, len(creds))
	for _, c := range creds {
		out = append(out, CredentialFromDomain(c))
	}
	return out
}

// maskAPIKey masks all but the last 4 characters of an API key.
func maskAPIKey(key string) string {
	if len(key) <= 4 {
		return strings.Repeat("*", len(key))
	}
	return strings.Repeat("*", len(key)-4) + key[len(key)-4:]
}
