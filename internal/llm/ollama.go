package llm

// NewOllama creates an OpenAI-compatible provider pointing at a local Ollama instance.
// The baseURL should be the Ollama server address, e.g. "http://localhost:11434".
func NewOllama(baseURL string) *OpenAIProvider {
	return newOpenAICompatible("ollama", baseURL+"/v1", nil)
}
