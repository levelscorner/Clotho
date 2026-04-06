package llm

// modelCost stores the per-1M-token cost for a model (input, output).
type modelCost struct {
	InputPerMillion  float64
	OutputPerMillion float64
}

// costTable maps model names to their pricing.
var costTable = map[string]modelCost{
	// OpenAI
	"gpt-4o":            {InputPerMillion: 2.50, OutputPerMillion: 10.00},
	"gpt-4o-2024-05-13": {InputPerMillion: 2.50, OutputPerMillion: 10.00},
	"gpt-4o-2024-08-06": {InputPerMillion: 2.50, OutputPerMillion: 10.00},
	"gpt-4o-mini":       {InputPerMillion: 0.15, OutputPerMillion: 0.60},
	"gpt-3.5-turbo":     {InputPerMillion: 0.50, OutputPerMillion: 1.50},

	// Gemini
	"gemini-2.0-flash": {InputPerMillion: 0, OutputPerMillion: 0},
	"gemini-1.5-pro":   {InputPerMillion: 1.25, OutputPerMillion: 5.00},
	"gemini-1.5-flash": {InputPerMillion: 0.075, OutputPerMillion: 0.30},

	// OpenRouter
	"anthropic/claude-sonnet-4": {InputPerMillion: 3.00, OutputPerMillion: 15.00},
	"meta-llama/llama-3-70b":    {InputPerMillion: 0.10, OutputPerMillion: 0.10},

	// Ollama (local, all free)
	"llama3":  {InputPerMillion: 0, OutputPerMillion: 0},
	"mistral": {InputPerMillion: 0, OutputPerMillion: 0},
	"phi3":    {InputPerMillion: 0, OutputPerMillion: 0},
	"gemma2":  {InputPerMillion: 0, OutputPerMillion: 0},
}

// CalculateCost returns the estimated cost in USD for the given model and usage.
// Returns 0 if the model is not in the cost table.
func CalculateCost(model string, usage TokenUsage) float64 {
	mc, ok := costTable[model]
	if !ok {
		return 0
	}
	inputCost := float64(usage.PromptTokens) / 1_000_000 * mc.InputPerMillion
	outputCost := float64(usage.CompletionTokens) / 1_000_000 * mc.OutputPerMillion
	return inputCost + outputCost
}
