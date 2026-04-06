package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/domain"
)

// CreateProjectRequest is the request body for creating a project.
type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateProjectRequest is the request body for updating a project.
type UpdateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ProjectResponse is the API response for a project.
type ProjectResponse struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProjectFromDomain converts a domain.Project to ProjectResponse.
func ProjectFromDomain(p domain.Project) ProjectResponse {
	return ProjectResponse{
		ID:          p.ID,
		TenantID:    p.TenantID,
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// ProjectsFromDomain converts a slice of domain.Project to ProjectResponse slice.
func ProjectsFromDomain(projects []domain.Project) []ProjectResponse {
	out := make([]ProjectResponse, 0, len(projects))
	for _, p := range projects {
		out = append(out, ProjectFromDomain(p))
	}
	return out
}
