package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
		tenantID := TenantIDFromContext(ctx)
		cred, credErr := e.credentials.Get(ctx, credID, tenantID)
		if credErr != nil {
			return StepOutput{}, fmt.Errorf("agent executor: load credential: %w", credErr)
		}
		apiKey, decErr := e.credentials.GetDecrypted(ctx, credID, tenantID)
		if decErr != nil {
			return StepOutput{}, fmt.Errorf("agent executor: decrypt credential: %w", decErr)
		}
		provider, providerErr = createProviderFromCredential(cred.Provider, apiKey)
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

	if cfg.CostCap != nil && costUSD > *cfg.CostCap {
		return StepOutput{}, fmt.Errorf("agent executor: cost cap exceeded: $%.4f > $%.4f cap", costUSD, *cfg.CostCap)
	}

	return StepOutput{
		Data:       json.RawMessage(outputData),
		TokensUsed: &tokensUsed,
		CostUSD:    &costUSD,
	}, nil
}

// resolveProvider extracts common prompt-building and provider resolution logic.
func (e *AgentExecutor) resolveProvider(ctx context.Context, cfg domain.AgentNodeConfig) (llm.Provider, llm.CompletionRequest, error) {
	systemPrompt := cfg.Role.SystemPrompt
	if cfg.Role.Persona != "" {
		systemPrompt = systemPrompt + "\n\nPersona: " + cfg.Role.Persona
	}

	model := cfg.Model
	if model == "" {
		model = "gemini-2.0-flash"
	}
	temperature := cfg.Temperature
	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}

	var (
		provider    llm.Provider
		providerErr error
	)
	if cfg.CredentialID != "" {
		credID, parseErr := uuid.Parse(cfg.CredentialID)
		if parseErr != nil {
			return nil, llm.CompletionRequest{}, fmt.Errorf("agent executor: invalid credential_id: %w", parseErr)
		}
		tenantID := TenantIDFromContext(ctx)
		cred, credErr := e.credentials.Get(ctx, credID, tenantID)
		if credErr != nil {
			return nil, llm.CompletionRequest{}, fmt.Errorf("agent executor: load credential: %w", credErr)
		}
		apiKey, decErr := e.credentials.GetDecrypted(ctx, credID, tenantID)
		if decErr != nil {
			return nil, llm.CompletionRequest{}, fmt.Errorf("agent executor: decrypt credential: %w", decErr)
		}
		provider, providerErr = createProviderFromCredential(cred.Provider, apiKey)
	} else {
		providerName := cfg.Provider
		if providerName == "" {
			providerName = "gemini"
		}
		provider, providerErr = e.providers.Get(providerName)
	}
	if providerErr != nil {
		return nil, llm.CompletionRequest{}, fmt.Errorf("agent executor: %w", providerErr)
	}

	req := llm.CompletionRequest{
		Model:        model,
		SystemPrompt: systemPrompt,
		Temperature:  temperature,
		MaxTokens:    maxTokens,
	}
	return provider, req, nil
}

// ExecuteStream runs an agent node with streaming output.
func (e *AgentExecutor) ExecuteStream(ctx context.Context, node domain.NodeInstance, inputs map[string]json.RawMessage) (<-chan ExecutorStreamChunk, <-chan StepOutput, <-chan error) {
	chunks := make(chan ExecutorStreamChunk, 64)
	result := make(chan StepOutput, 1)
	errCh := make(chan error, 1)

	go func() {
		// Only close the chunks channel — the engine iterates it with
		// `for chunk := range chunks` and needs the close to exit the
		// loop. Closing result AND errCh on top of that would make the
		// engine's downstream `select { case <-result; case <-errCh }`
		// race: both closed channels are "ready" simultaneously, so Go
		// picks one non-deterministically. If errCh wins on a success
		// path, the engine reads execErr=nil AND stepOut=zero (Data=nil)
		// and treats the node as successful-with-no-output — the exact
		// symptom seen in production when step_results.output_data
		// landed NULL after a 22-second Ollama run.
		//
		// result and errCh stay open — we always send on exactly one
		// of them before returning, and the goroutine exit causes Go
		// to GC the channels once the engine drops its references.
		defer close(chunks)

		var cfg domain.AgentNodeConfig
		if err := json.Unmarshal(node.Config, &cfg); err != nil {
			errCh <- fmt.Errorf("agent executor: unmarshal config: %w", err)
			return
		}

		provider, req, err := e.resolveProvider(ctx, cfg)
		if err != nil {
			errCh <- err
			return
		}

		userPrompt := buildUserPrompt(cfg.Task.Template, inputs)
		req.UserPrompt = userPrompt

		stream, err := provider.Stream(ctx, req)
		if err != nil {
			errCh <- fmt.Errorf("agent executor: llm stream: %w", err)
			return
		}

		var fullContent strings.Builder
		for chunk := range stream {
			if chunk.Content != "" {
				fullContent.WriteString(chunk.Content)
				chunks <- ExecutorStreamChunk{Content: chunk.Content}
			}
		}

		content := fullContent.String()
		outputData, err := json.Marshal(content)
		if err != nil {
			errCh <- fmt.Errorf("agent executor: marshal output: %w", err)
			return
		}

		estTokens := len(content) / 4 // rough estimate for streaming
		costUSD := llm.CalculateCost(req.Model, llm.TokenUsage{
			CompletionTokens: estTokens,
			TotalTokens:      estTokens,
		})
		tokensUsed := estTokens

		if cfg.CostCap != nil && costUSD > *cfg.CostCap {
			errCh <- fmt.Errorf("agent executor: cost cap exceeded: $%.4f > $%.4f cap", costUSD, *cfg.CostCap)
			return
		}

		result <- StepOutput{
			Data:       json.RawMessage(outputData),
			TokensUsed: &tokensUsed,
			CostUSD:    &costUSD,
		}
	}()

	return chunks, result, errCh
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

	// Sort keys for deterministic prompt assembly (Go map iteration is random).
	keys := make([]string, 0, len(inputs))
	for k := range inputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		raw := inputs[k]
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
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
