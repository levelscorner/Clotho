package engine

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/user/clotho/internal/storage"
)

// fakeStore is a minimal in-test storage.Store implementation. It captures
// the arguments passed to Write so tests can assert on them without reaching
// for a real filesystem. Do NOT depend on the real LocalStore — keeping the
// test isolated makes it fast and portable.
type fakeStore struct {
	gotLoc      storage.Location
	gotFilename string
	gotData     []byte
	writeErr    error
}

func (f *fakeStore) Write(_ context.Context, loc storage.Location, filename string, data []byte) (string, string, error) {
	if f.writeErr != nil {
		return "", "", f.writeErr
	}
	f.gotLoc = loc
	f.gotFilename = filename
	// Copy so later mutations by the caller don't affect what we captured.
	buf := make([]byte, len(data))
	copy(buf, data)
	f.gotData = buf
	rel := loc.ProjectSlug + "/" + loc.PipelineSlug + "/" + loc.ExecutionID.String() + "/" + filename
	abs := "/fake/base/" + rel
	return rel, abs, nil
}

func (f *fakeStore) Read(_ context.Context, _ string) (io.ReadCloser, string, error) {
	return nil, "", errors.New("not implemented")
}

func (f *fakeStore) Base() string { return "/fake/base" }

func newSampleManifest() Manifest {
	return Manifest{
		ExecutionID:  uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		PipelineID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		PipelineName: "demo pipeline",
		ProjectID:    uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		StartedAt:    time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC),
		CompletedAt:  time.Date(2026, 4, 14, 10, 1, 30, 0, time.UTC),
		TotalCostUSD: 0.0125,
		TotalTokens:  1234,
		Nodes: []ManifestNode{
			{
				NodeID:     "node-1",
				NodeName:   "Script Writer",
				Type:       "agent",
				Provider:   "openai",
				Model:      "gpt-4o-mini",
				Prompt:     "Write a dramatic scene",
				Output:     "Scene opens in a dim studio...",
				DurationMs: 2500,
				CostUSD:    0.0025,
				TokensUsed: 420,
				Status:     "completed",
			},
			{
				NodeID:     "node-2",
				NodeName:   "Hero Image",
				Type:       "media",
				Provider:   "comfyui",
				Model:      "flux1-schnell",
				Prompt:     "Dim studio at night",
				OutputFile: "image-abc.png",
				DurationMs: 71000,
				Status:     "completed",
			},
			{
				NodeID: "node-3",
				Type:   "agent",
				Status: "failed",
				Error:  "provider timed out",
			},
		},
	}
}

func TestWriteManifest_WritesJSONViaStore(t *testing.T) {
	store := &fakeStore{}
	loc := storage.Location{
		ProjectID:    uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		ProjectSlug:  "demo-project",
		PipelineID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		PipelineSlug: "demo-pipeline",
		ExecutionID:  uuid.MustParse("11111111-1111-1111-1111-111111111111"),
	}

	m := newSampleManifest()
	rel, err := WriteManifest(context.Background(), store, loc, m)
	if err != nil {
		t.Fatalf("WriteManifest returned error: %v", err)
	}
	if rel == "" {
		t.Fatal("expected non-empty relative path")
	}
	if !strings.HasSuffix(rel, "/manifest.json") {
		t.Errorf("relative path = %q, want suffix /manifest.json", rel)
	}

	if store.gotFilename != "manifest.json" {
		t.Errorf("filename = %q, want manifest.json", store.gotFilename)
	}
	if store.gotLoc != loc {
		t.Errorf("Location = %+v, want %+v", store.gotLoc, loc)
	}

	// Verify pretty-printing (two-space indent) is used.
	body := string(store.gotData)
	if !strings.Contains(body, "\n  \"execution_id\"") {
		t.Errorf("expected indented JSON, got: %s", body)
	}
}

func TestWriteManifest_OmitsEmptyOptionalFields(t *testing.T) {
	store := &fakeStore{}
	m := Manifest{
		ExecutionID:  uuid.New(),
		PipelineID:   uuid.New(),
		PipelineName: "p",
		ProjectID:    uuid.New(),
		StartedAt:    time.Now(),
		CompletedAt:  time.Now(),
		Nodes: []ManifestNode{
			{NodeID: "n1", Type: "agent", Status: "completed"},
		},
	}

	_, err := WriteManifest(context.Background(), store, storage.Location{}, m)
	if err != nil {
		t.Fatalf("WriteManifest returned error: %v", err)
	}

	body := string(store.gotData)
	// Node n1 has no output_file / output / provider / model / prompt / error.
	forbidden := []string{
		`"output_file"`,
		`"output"`,
		`"provider"`,
		`"model"`,
		`"prompt"`,
		`"error"`,
		`"node_name"`,
	}
	for _, f := range forbidden {
		if strings.Contains(body, f) {
			t.Errorf("expected %s to be omitted via omitempty, got: %s", f, body)
		}
	}
}

func TestWriteManifest_RoundTrip(t *testing.T) {
	store := &fakeStore{}
	original := newSampleManifest()

	if _, err := WriteManifest(context.Background(), store, storage.Location{}, original); err != nil {
		t.Fatalf("WriteManifest returned error: %v", err)
	}

	var decoded Manifest
	if err := json.Unmarshal(store.gotData, &decoded); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	if decoded.ExecutionID != original.ExecutionID {
		t.Errorf("ExecutionID round-trip mismatch: got %v, want %v", decoded.ExecutionID, original.ExecutionID)
	}
	if decoded.PipelineName != original.PipelineName {
		t.Errorf("PipelineName round-trip mismatch: got %q, want %q", decoded.PipelineName, original.PipelineName)
	}
	if decoded.TotalCostUSD != original.TotalCostUSD {
		t.Errorf("TotalCostUSD round-trip mismatch: got %v, want %v", decoded.TotalCostUSD, original.TotalCostUSD)
	}
	if len(decoded.Nodes) != len(original.Nodes) {
		t.Fatalf("Nodes length mismatch: got %d, want %d", len(decoded.Nodes), len(original.Nodes))
	}
	if decoded.Nodes[1].OutputFile != "image-abc.png" {
		t.Errorf("Nodes[1].OutputFile = %q, want %q", decoded.Nodes[1].OutputFile, "image-abc.png")
	}
	if decoded.Nodes[2].Error != "provider timed out" {
		t.Errorf("Nodes[2].Error = %q, want %q", decoded.Nodes[2].Error, "provider timed out")
	}
	if !decoded.StartedAt.Equal(original.StartedAt) {
		t.Errorf("StartedAt round-trip mismatch: got %v, want %v", decoded.StartedAt, original.StartedAt)
	}
}

func TestWriteManifest_NilStoreError(t *testing.T) {
	_, err := WriteManifest(context.Background(), nil, storage.Location{}, newSampleManifest())
	if err == nil {
		t.Fatal("expected error for nil store, got nil")
	}
}

func TestWriteManifest_PropagatesStoreError(t *testing.T) {
	store := &fakeStore{writeErr: errors.New("disk full")}
	_, err := WriteManifest(context.Background(), store, storage.Location{}, newSampleManifest())
	if err == nil {
		t.Fatal("expected error from store, got nil")
	}
	if !strings.Contains(err.Error(), "disk full") {
		t.Errorf("error = %q, want to include %q", err.Error(), "disk full")
	}
}
