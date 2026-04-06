package llm

import (
	"math"
	"testing"
)

func TestCalculateCost(t *testing.T) {
	t.Parallel()

	t.Run("known model gpt-4o", func(t *testing.T) {
		t.Parallel()
		usage := TokenUsage{PromptTokens: 1000, CompletionTokens: 500}
		got := CalculateCost("gpt-4o", usage)
		// input: 1000/1M * 2.50 = 0.0025
		// output: 500/1M * 10.00 = 0.005
		want := 0.0075
		if math.Abs(got-want) > 1e-10 {
			t.Errorf("CalculateCost('gpt-4o', ...) = %v, want %v", got, want)
		}
	})

	t.Run("unknown model returns 0", func(t *testing.T) {
		t.Parallel()
		usage := TokenUsage{PromptTokens: 1000, CompletionTokens: 500}
		got := CalculateCost("nonexistent-model", usage)
		if got != 0 {
			t.Errorf("CalculateCost('nonexistent-model', ...) = %v, want 0", got)
		}
	})

	t.Run("zero tokens returns 0", func(t *testing.T) {
		t.Parallel()
		usage := TokenUsage{PromptTokens: 0, CompletionTokens: 0}
		got := CalculateCost("gpt-4o", usage)
		if got != 0 {
			t.Errorf("CalculateCost('gpt-4o', zero tokens) = %v, want 0", got)
		}
	})

	t.Run("large token count", func(t *testing.T) {
		t.Parallel()
		usage := TokenUsage{PromptTokens: 1_000_000, CompletionTokens: 1_000_000}
		got := CalculateCost("gpt-4o", usage)
		// input: 1M/1M * 2.50 = 2.50
		// output: 1M/1M * 10.00 = 10.00
		want := 12.50
		if math.Abs(got-want) > 1e-10 {
			t.Errorf("CalculateCost('gpt-4o', 1M tokens) = %v, want %v", got, want)
		}
	})

	// Verify every model in the cost table returns non-zero for non-zero tokens
	// (except free models which correctly return 0)
	t.Run("all models in cost table", func(t *testing.T) {
		t.Parallel()

		paidModels := []struct {
			name string
		}{
			{"gpt-4o"},
			{"gpt-4o-2024-05-13"},
			{"gpt-4o-2024-08-06"},
			{"gpt-4o-mini"},
			{"gpt-3.5-turbo"},
			{"gemini-1.5-pro"},
			{"gemini-1.5-flash"},
			{"anthropic/claude-sonnet-4"},
			{"meta-llama/llama-3-70b"},
		}

		usage := TokenUsage{PromptTokens: 1000, CompletionTokens: 1000}
		for _, m := range paidModels {
			t.Run(m.name, func(t *testing.T) {
				t.Parallel()
				got := CalculateCost(m.name, usage)
				if got <= 0 {
					t.Errorf("CalculateCost(%q, ...) = %v, expected > 0 for paid model", m.name, got)
				}
			})
		}

		freeModels := []string{"gemini-2.0-flash", "llama3", "mistral", "phi3", "gemma2"}
		for _, name := range freeModels {
			t.Run(name+"_free", func(t *testing.T) {
				t.Parallel()
				got := CalculateCost(name, usage)
				if got != 0 {
					t.Errorf("CalculateCost(%q, ...) = %v, expected 0 for free model", name, got)
				}
			})
		}
	})
}
