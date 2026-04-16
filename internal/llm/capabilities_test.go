package llm

import (
	"sort"
	"testing"
)

// TestCapabilityTableCompleteness asserts every known provider has an entry
// for every capability. This is the backstop against a provider being added
// to KnownProviders() but forgotten in capabilityTable.
func TestCapabilityTableCompleteness(t *testing.T) {
	expectedProviders := []string{"openai", "gemini", "openrouter", "ollama"}
	actual := KnownProviders()
	sort.Strings(actual)
	sort.Strings(expectedProviders)

	if len(actual) != len(expectedProviders) {
		t.Fatalf("expected %d providers, got %d: %v", len(expectedProviders), len(actual), actual)
	}
	for i, p := range expectedProviders {
		if actual[i] != p {
			t.Fatalf("expected provider %q at index %d, got %q", p, i, actual[i])
		}
	}
}

func TestAppliesTo(t *testing.T) {
	tests := []struct {
		provider string
		cap      Capability
		want     bool
	}{
		{"openai", CapTopK, false},
		{"openai", CapSeed, true},
		{"openai", CapFrequencyPenalty, true},
		{"openai", CapTools, true},

		{"gemini", CapTopK, true},
		{"gemini", CapSeed, true},
		{"gemini", CapJSONSchema, true},

		{"ollama", CapTopK, true},
		{"ollama", CapFrequencyPenalty, false},
		{"ollama", CapSeed, true},

		{"openrouter", CapTopK, true},
		{"openrouter", CapTools, true},

		// Unknown provider: every capability false.
		{"anthropic", CapTopK, false},
		{"anthropic", CapTools, false},
		{"", CapSeed, false},
	}

	for _, tc := range tests {
		t.Run(tc.provider+"/"+string(tc.cap), func(t *testing.T) {
			got := AppliesTo(tc.provider, tc.cap)
			if got != tc.want {
				t.Errorf("AppliesTo(%q, %q) = %v, want %v", tc.provider, tc.cap, got, tc.want)
			}
		})
	}
}

func TestCapabilitiesForUnknown(t *testing.T) {
	caps := CapabilitiesFor("does-not-exist")
	// Zero value should be all-false.
	if caps.TopK || caps.Seed || caps.FrequencyPenalty || caps.PresencePenalty ||
		caps.StopSequences || caps.JSONMode || caps.JSONSchema || caps.Tools {
		t.Errorf("unknown provider should return zero-value capabilities, got %+v", caps)
	}
}
