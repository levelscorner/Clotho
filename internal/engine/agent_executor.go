package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/llm"
	"github.com/user/clotho/internal/store"
)

// AgentExecutor implements StepExecutor for agent nodes.
type AgentExecutor struct {
	providers   *llm.ProviderRegistry
	credentials store.CredentialStore
}

// NewAgentExecutor creates an AgentExecutor with a provider registry and credential store.
func NewAgentExecutor(providers *llm.ProviderRegistry, credentials store.CredentialStore) *AgentExecutor {
	return &AgentExecutor{
		providers:   providers,
		credentials: credentials,
	}
}

// Execute runs an agent node: builds prompts from config, calls the LLM, and returns output.
func (e *AgentExecutor) Execute(ctx context.Context, node domain.NodeInstance, inputs map[string]json.RawMessage) (StepOutput, error) {
	var cfg domain.AgentNodeConfig
	if err := json.Unmarshal(node.Config, &cfg); err != nil {
		return StepOutput{}, fmt.Errorf("agent executor: unmarshal config: %w", err)
	}

	// Build system prompt from role config
	systemPrompt := cfg.Role.SystemPrompt
	if cfg.Role.Persona != "" {
		systemPrompt = systemPrompt + "\n\nPersona: " + cfg.Role.Persona
	}

	// Build user prompt from task template + inputs
	userPrompt := buildUserPrompt(cfg.Task.Template, inputs)

	model := cfg.Model
	if model == "" {
		model = "gemini-2.0-flash"
	}

	temperature := cfg.Temperature
	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}

	// Resolve provider: credential-based takes precedence, then registry lookup
	var (
		provider    llm.Provider
		providerErr error
	)
	if cfg.CredentialID != "" {
		credID, parseErr := uuid.Parse(cfg.CredentialID)
		if parseErr != nil {
			return StepOutput{}, fmt.Errorf("agent executor: invalid credential_id: %w", parseErr)
		}
		cred, credErr := e.credentials.Get(ctx, credID)
		if credErr != nil {
			return StepOutput{}, fmt.Errorf("agent executor: load credential: %w", credErr)
		}
		provider, providerErr = createProviderFromCredential(cred.Provider, cred.APIKey)
	} else {
		providerName := cfg.Provider
		if providerName == "" {
			providerName = "gemini"
		}
		provider, providerErr = e.providers.Get(providerName)
	}
	if providerErr != nil {
		return StepOutput{}, fmt.Errorf("agent executor: %w", providerErr)
	}

	resp, err := provider.Complete(ctx, llm.CompletionRequest{
		Model:        model,
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  temperature,
		MaxTokens:    maxTokens,
	})
	if err != nil {
		return StepOutput{}, fmt.Errorf("agent executor: llm complete: %w", err)
	}

	outputData, err := json.Marshal(resp.Content)
	if err != nil {
		return StepOutput{}, fmt.Errorf("agent executor: marshal output: %w", err)
	}

	tokensUsed := resp.Usage.TotalTokens
	costUSD := resp.CostUSD

	return StepOutput{
		Data:       json.RawMessage(outputData),
		TokensUsed: &tokensUsed,
		CostUSD:    &costUSD,
	}, nil
}

// buildUserPrompt renders the task template with input data.
// If template contains {{input}}, it is replaced with concatenated input values.
// If template is empty, inputs are concatenated directly.
func buildUserPrompt(template string, inputs map[string]json.RawMessage) string {
	inputText := concatenateInputs(inputs)

	if template == "" {
		return inputText
	}

	if strings.Contains(template, "{{input}}") {
		return strings.ReplaceAll(template, "{{input}}", inputText)
	}

	// Template exists but no placeholder: append inputs after template
	if inputText != "" {
		return template + "\n\n" + inputText
	}
	return template
}

// concatenateInputs joins all input values into a single string.
func concatenateInputs(inputs map[string]json.RawMessage) string {
	if len(inputs) == 0 {
		return ""
	}

	var parts []string
	for _, raw := range inputs {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			// If not a JSON string, use the raw bytes
			parts = append(parts, string(raw))
		} else {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "\n")
}

// createProviderFromCredential creates an LLM provider on-the-fly from a stored credential.
func createProviderFromCredential(providerName, apiKey string) (llm.Provider, error) {
	switch providerName {
	case "openai":
		return llm.NewOpenAI(apiKey), nil
	case "gemini":
		return llm.NewGemini(apiKey), nil
	case "openrouter":
		return llm.NewOpenRouter(apiKey), nil
	default:
		return nil, fmt.Errorf("unsupported credential provider: %s", providerName)
	}
}
