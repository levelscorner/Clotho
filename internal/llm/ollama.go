package llm

// NewOllama creates an OpenAI-compatible provider pointing at a local Ollama instance.
// The baseURL should be the Ollama server address, e.g. "http://localhost:11434".
// The first argument to newOpenAICompatible is the API key; Ollama treats it
// as a bearer stub so the literal string "ollama" works.
func NewOllama(baseURL string) *OpenAIProvider {
	return newOpenAICompatible("ollama", baseURL+"/v1", "ollama", nil)
}
