package domain

import (
	"time"

	"github.com/google/uuid"
)

// Pipeline is a named workflow within a project.
type Pipeline struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Viewport represents the canvas view state (zoom + pan).
type Viewport struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Zoom float64 `json:"zoom"`
}

// PipelineGraph is the full graph definition: nodes, edges, viewport.
// Stored as JSONB in pipeline_versions.
type PipelineGraph struct {
	Nodes    []NodeInstance `json:"nodes"`
	Edges    []Edge         `json:"edges"`
	Viewport Viewport       `json:"viewport"`
}

// PipelineVersion is an immutable snapshot of a pipeline graph.
type PipelineVersion struct {
	ID         uuid.UUID     `json:"id"`
	PipelineID uuid.UUID     `json:"pipeline_id"`
	Version    int           `json:"version"`
	Graph      PipelineGraph `json:"graph"`
	CreatedAt  time.Time     `json:"created_at"`
}
