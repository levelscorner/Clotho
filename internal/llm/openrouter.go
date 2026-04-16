package llm

// NewOpenRouter creates an OpenAI-compatible provider pointing at the OpenRouter API.
func NewOpenRouter(apiKey string) *OpenAIProvider {
	return newOpenAICompatible(apiKey, "https://openrouter.ai/api/v1", "openrouter", map[string]string{
		"HTTP-Referer": "https://clotho.app",
		"X-Title":      "Clotho",
	})
}
