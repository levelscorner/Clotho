package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/domain"
)

// CreatePipelineRequest is the request body for creating a pipeline.
type CreatePipelineRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdatePipelineRequest is the request body for updating a pipeline.
type UpdatePipelineRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SaveVersionRequest is the request body for saving a new pipeline version.
type SaveVersionRequest struct {
	Graph domain.PipelineGraph `json:"graph"`
}

// PipelineResponse is the API response for a pipeline.
type PipelineResponse struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PipelineVersionResponse is the API response for a pipeline version.
type PipelineVersionResponse struct {
	ID         uuid.UUID            `json:"id"`
	PipelineID uuid.UUID            `json:"pipeline_id"`
	Version    int                  `json:"version"`
	Graph      domain.PipelineGraph `json:"graph"`
	CreatedAt  time.Time            `json:"created_at"`
}

// PipelineFromDomain converts a domain.Pipeline to PipelineResponse.
func PipelineFromDomain(p domain.Pipeline) PipelineResponse {
	return PipelineResponse{
		ID:          p.ID,
		ProjectID:   p.ProjectID,
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// PipelinesFromDomain converts a slice of domain.Pipeline to PipelineResponse slice.
func PipelinesFromDomain(pipelines []domain.Pipeline) []PipelineResponse {
	out := make([]PipelineResponse, 0, len(pipelines))
	for _, p := range pipelines {
		out = append(out, PipelineFromDomain(p))
	}
	return out
}

// PipelineVersionFromDomain converts a domain.PipelineVersion to PipelineVersionResponse.
func PipelineVersionFromDomain(pv domain.PipelineVersion) PipelineVersionResponse {
	return PipelineVersionResponse{
		ID:         pv.ID,
		PipelineID: pv.PipelineID,
		Version:    pv.Version,
		Graph:      pv.Graph,
		CreatedAt:  pv.CreatedAt,
	}
}
