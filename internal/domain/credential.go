package domain

import (
	"time"

	"github.com/google/uuid"
)

// Credential stores an encrypted API key for an LLM provider.
type Credential struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	Provider       string    `json:"provider"` // "openai", "anthropic", "gemini"
	EncryptedValue []byte    `json:"-"`
	EncryptedDEK   []byte    `json:"-"`
	Nonce          []byte    `json:"-"`
	Label          string    `json:"label"`
	CreatedAt      time.Time `json:"created_at"`

	// PlaintextKey is transient; only populated in memory, never persisted directly.
	PlaintextKey string `json:"-"`
}
