package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/user/clotho/internal/domain"
)

func agentNodeWithTimeout(t *testing.T, sec *int) domain.NodeInstance {
	t.Helper()
	cfg := domain.AgentNodeConfig{
		Provider:       "openai",
		Model:          "gpt-4o",
		StepTimeoutSec: sec,
	}
	bs, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal cfg: %v", err)
	}
	return domain.NodeInstance{
		ID:     "n1",
		Type:   domain.NodeTypeAgent,
		Config: bs,
	}
}

func TestStepTimeoutFor_AgentRespectsOverride(t *testing.T) {
	t.Parallel()
	override := 30
	got := stepTimeoutFor(agentNodeWithTimeout(t, &override))
	want := 30 * time.Second
	if got != want {
		t.Errorf("stepTimeoutFor(override=30s) = %v, want %v", got, want)
	}
}

func TestStepTimeoutFor_AgentDefaultsWhenUnset(t *testing.T) {
	t.Parallel()
	got := stepTimeoutFor(agentNodeWithTimeout(t, nil))
	if got != DefaultStepTimeout {
		t.Errorf("stepTimeoutFor(nil) = %v, want default %v", got, DefaultStepTimeout)
	}
}

func TestStepTimeoutFor_AgentDefaultsWhenZeroOrNegative(t *testing.T) {
	t.Parallel()
	zero := 0
	got := stepTimeoutFor(agentNodeWithTimeout(t, &zero))
	if got != DefaultStepTimeout {
		t.Errorf("zero override should fall back to default; got %v", got)
	}
	neg := -5
	got = stepTimeoutFor(agentNodeWithTimeout(t, &neg))
	if got != DefaultStepTimeout {
		t.Errorf("negative override should fall back to default; got %v", got)
	}
}

func TestStepTimeoutFor_NonAgentUsesDefault(t *testing.T) {
	t.Parallel()
	for _, nt := range []domain.NodeType{domain.NodeTypeTool, domain.NodeTypeMedia} {
		node := domain.NodeInstance{Type: nt, Config: json.RawMessage(`{}`)}
		got := stepTimeoutFor(node)
		if got != DefaultStepTimeout {
			t.Errorf("%s node should use default; got %v", nt, got)
		}
	}
}

func TestStepTimeoutFor_MalformedConfigFallsBackToDefault(t *testing.T) {
	t.Parallel()
	node := domain.NodeInstance{
		Type:   domain.NodeTypeAgent,
		Config: json.RawMessage(`not valid json`),
	}
	got := stepTimeoutFor(node)
	if got != DefaultStepTimeout {
		t.Errorf("malformed config should not panic; got %v", got)
	}
}
