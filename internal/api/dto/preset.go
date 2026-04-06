package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/domain"
)

// CreatePresetRequest is the request body for creating a custom preset.
type CreatePresetRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Config      domain.AgentNodeConfig `json:"config"`
	Icon        string                 `json:"icon"`
}

// UpdatePresetRequest is the request body for updating a preset.
type UpdatePresetRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Config      domain.AgentNodeConfig `json:"config"`
	Icon        string                 `json:"icon"`
}

// PresetResponse is the API response for a preset.
type PresetResponse struct {
	ID          uuid.UUID              `json:"id"`
	TenantID    *uuid.UUID             `json:"tenant_id,omitempty"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Config      domain.AgentNodeConfig `json:"config"`
	Icon        string                 `json:"icon"`
	IsBuiltIn   bool                   `json:"is_built_in"`
	CreatedAt   time.Time              `json:"created_at"`
}

// PresetFromDomain converts a domain.AgentPreset to PresetResponse.
func PresetFromDomain(p domain.AgentPreset) PresetResponse {
	return PresetResponse{
		ID:          p.ID,
		TenantID:    p.TenantID,
		Name:        p.Name,
		Description: p.Description,
		Category:    p.Category,
		Config:      p.Config,
		Icon:        p.Icon,
		IsBuiltIn:   p.IsBuiltIn,
		CreatedAt:   p.CreatedAt,
	}
}

// PresetsFromDomain converts a slice of domain.AgentPreset to PresetResponse slice.
func PresetsFromDomain(presets []domain.AgentPreset) []PresetResponse {
	out := make([]PresetResponse, 0, len(presets))
	for _, p := range presets {
		out = append(out, PresetFromDomain(p))
	}
	return out
}
