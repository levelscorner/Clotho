package llm

import (
	"context"
	"testing"
)

// mockProvider implements Provider for testing.
type mockProvider struct {
	name string
}

func (m *mockProvider) Complete(_ context.Context, _ CompletionRequest) (CompletionResponse, error) {
	return CompletionResponse{Content: m.name}, nil
}

func (m *mockProvider) Stream(_ context.Context, _ CompletionRequest) (<-chan StreamChunk, error) {
	return nil, nil
}

func TestProviderRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	p := &mockProvider{name: "test-openai"}
	reg.Register("openai", p)

	got, err := reg.Get("openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != p {
		t.Error("Get returned different provider than registered")
	}
}

func TestProviderRegistry_GetUnregistered(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistered provider, got nil")
	}
}

func TestProviderRegistry_List(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	reg.Register("zebra", &mockProvider{name: "z"})
	reg.Register("alpha", &mockProvider{name: "a"})
	reg.Register("middle", &mockProvider{name: "m"})

	names := reg.List()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "alpha" {
		t.Errorf("names[0] = %q, want %q", names[0], "alpha")
	}
	if names[1] != "middle" {
		t.Errorf("names[1] = %q, want %q", names[1], "middle")
	}
	if names[2] != "zebra" {
		t.Errorf("names[2] = %q, want %q", names[2], "zebra")
	}
}

func TestProviderRegistry_RegisterOverwrites(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	p1 := &mockProvider{name: "first"}
	p2 := &mockProvider{name: "second"}

	reg.Register("openai", p1)
	reg.Register("openai", p2)

	got, err := reg.Get("openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != p2 {
		t.Error("expected overwritten provider, got original")
	}
}

func TestProviderRegistry_ListEmpty(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	names := reg.List()
	if len(names) != 0 {
		t.Errorf("expected 0 names, got %d", len(names))
	}
}
