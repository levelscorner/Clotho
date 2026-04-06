package domain

import (
	"time"

	"github.com/google/uuid"
)

// AgentPreset is a reusable agent configuration template.
// The 7 built-in presets from Scribble.md are system presets (IsBuiltIn=true).
type AgentPreset struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    *uuid.UUID      `json:"tenant_id,omitempty"` // nil for built-in system presets
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Config      AgentNodeConfig `json:"config"`
	Icon        string          `json:"icon"`
	IsBuiltIn   bool            `json:"is_built_in"`
	CreatedAt   time.Time       `json:"created_at"`
}
