package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/user/clotho/internal/storage"
)

// Manifest is the JSON sidecar written alongside each execution's output
// files. Dropped into Finder, it lets a human (or future tool) understand
// the whole run — prompts, outputs, timings, cost, errors — without
// re-fetching anything from the database.
type Manifest struct {
	ExecutionID  uuid.UUID      `json:"execution_id"`
	PipelineID   uuid.UUID      `json:"pipeline_id"`
	PipelineName string         `json:"pipeline_name"`
	ProjectID    uuid.UUID      `json:"project_id"`
	StartedAt    time.Time      `json:"started_at"`
	CompletedAt  time.Time      `json:"completed_at"`
	TotalCostUSD float64        `json:"total_cost_usd"`
	TotalTokens  int            `json:"total_tokens"`
	Nodes        []ManifestNode `json:"nodes"`
}

// ManifestNode captures a single node's contribution to a pipeline run. Most
// fields use omitempty so the serialised manifest stays readable when only
// part of the data applies (e.g. agent nodes have no OutputFile; media nodes
// have no inline Output text).
type ManifestNode struct {
	NodeID     string  `json:"node_id"`
	NodeName   string  `json:"node_name,omitempty"`
	Type       string  `json:"type"` // agent / media / tool
	Provider   string  `json:"provider,omitempty"`
	Model      string  `json:"model,omitempty"`
	Prompt     string  `json:"prompt,omitempty"`
	OutputFile string  `json:"output_file,omitempty"` // rel path inside execution dir, if on disk
	Output     string  `json:"output,omitempty"`      // inline text output for agent nodes
	DurationMs int64   `json:"duration_ms,omitempty"`
	CostUSD    float64 `json:"cost_usd,omitempty"`
	TokensUsed int     `json:"tokens_used,omitempty"`
	Status     string  `json:"status"`
	Error      string  `json:"error,omitempty"`
}

// WriteManifest serialises m to manifest.json under the given Location via
// the Store. It returns the relative path on success so the caller can log
// or persist it alongside the execution record.
func WriteManifest(ctx context.Context, store storage.Store, loc storage.Location, m Manifest) (string, error) {
	if store == nil {
		return "", fmt.Errorf("engine: manifest store is nil")
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", fmt.Errorf("engine: marshal manifest: %w", err)
	}

	rel, _, err := store.Write(ctx, loc, "manifest.json", data)
	if err != nil {
		return "", fmt.Errorf("engine: write manifest: %w", err)
	}
	return rel, nil
}
