package llm

// Capability is a single knob the Prompt Component inspector may surface.
// Provider adapters consult Capabilities at translate time to decide which
// request fields to drop silently.
type Capability string

const (
	CapTopK             Capability = "top_k"
	CapSeed             Capability = "seed"
	CapFrequencyPenalty Capability = "frequency_penalty"
	CapPresencePenalty  Capability = "presence_penalty"
	CapStopSequences    Capability = "stop_sequences"
	CapJSONMode         Capability = "json_mode"
	CapJSONSchema       Capability = "json_schema"
	CapTools            Capability = "tools"
)

// ProviderCapabilities records which knobs a provider honors. Model-level
// capabilities (vision, reasoning) are computed on the fly from the model
// name because they change per model within a single provider.
type ProviderCapabilities struct {
	TopK             bool
	Seed             bool
	FrequencyPenalty bool
	PresencePenalty  bool
	StopSequences    bool
	JSONMode         bool
	JSONSchema       bool
	Tools            bool
}

// capabilityTable is the source of truth for provider knob support. The
// frontend mirror in web/src/lib/llmCapabilities.ts must stay in sync —
// a table-driven test asserts every declared provider has all capability
// keys defined.
var capabilityTable = map[string]ProviderCapabilities{
	"openai": {
		TopK:             false,
		Seed:             true,
		FrequencyPenalty: true,
		PresencePenalty:  true,
		StopSequences:    true,
		JSONMode:         true,
		JSONSchema:       true,
		Tools:            true,
	},
	"gemini": {
		TopK:             true,
		Seed:             true,
		FrequencyPenalty: true,
		PresencePenalty:  true,
		StopSequences:    true,
		JSONMode:         true,
		JSONSchema:       true,
		Tools:            true,
	},
	"openrouter": {
		// OpenRouter forwards to whichever upstream model is selected.
		// Default to OpenAI-compatible semantics; the upstream may still
		// ignore specific fields silently.
		TopK:             true,
		Seed:             true,
		FrequencyPenalty: true,
		PresencePenalty:  true,
		StopSequences:    true,
		JSONMode:         true,
		JSONSchema:       true,
		Tools:            true,
	},
	"ollama": {
		TopK:             true,
		Seed:             true,
		FrequencyPenalty: false,
		PresencePenalty:  false,
		StopSequences:    true,
		JSONMode:         true,
		JSONSchema:       true,
		Tools:            true,
	},
}

// CapabilitiesFor returns the capabilities for a provider. Unknown providers
// return a zero-value struct (everything false) so callers drop every
// optional knob — safe default when we don't recognize the adapter.
func CapabilitiesFor(provider string) ProviderCapabilities {
	if caps, ok := capabilityTable[provider]; ok {
		return caps
	}
	return ProviderCapabilities{}
}

// AppliesTo reports whether a given capability is honored by the named
// provider. Provider adapters use this to decide what to forward.
func AppliesTo(provider string, cap Capability) bool {
	caps := CapabilitiesFor(provider)
	switch cap {
	case CapTopK:
		return caps.TopK
	case CapSeed:
		return caps.Seed
	case CapFrequencyPenalty:
		return caps.FrequencyPenalty
	case CapPresencePenalty:
		return caps.PresencePenalty
	case CapStopSequences:
		return caps.StopSequences
	case CapJSONMode:
		return caps.JSONMode
	case CapJSONSchema:
		return caps.JSONSchema
	case CapTools:
		return caps.Tools
	}
	return false
}

// KnownProviders returns the list of providers the capability table knows
// about. Used by tests to assert completeness.
func KnownProviders() []string {
	out := make([]string, 0, len(capabilityTable))
	for name := range capabilityTable {
		out = append(out, name)
	}
	return out
}
