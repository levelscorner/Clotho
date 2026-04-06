package domain

import (
	"time"

	"github.com/google/uuid"
)

// Credential stores an API key for an LLM provider.
// Phase 1: plaintext storage. Phase 2: envelope encryption.
type Credential struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Provider  string    `json:"provider"` // "openai", "anthropic", "gemini"
	APIKey    string    `json:"-"`        // never serialized to JSON
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
}
