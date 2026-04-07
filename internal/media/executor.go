package media

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/engine"
	"github.com/user/clotho/internal/store"
)

const (
	pollInitialInterval = 2 * time.Second
	pollMaxInterval     = 10 * time.Second
	pollTimeout         = 5 * time.Minute
)

// MediaExecutor implements engine.StepExecutor for media generation nodes.
type MediaExecutor struct {
	providers   *Registry
	credentials store.CredentialStore
}

// NewMediaExecutor creates a MediaExecutor with a provider registry and credential store.
func NewMediaExecutor(providers *Registry, credentials store.CredentialStore) *MediaExecutor {
	return &MediaExecutor{providers: providers, credentials: credentials}
}

// Execute runs a media node synchronously: submits a job, polls until complete, and returns the output URL.
func (e *MediaExecutor) Execute(ctx context.Context, node domain.NodeInstance, inputs map[string]json.RawMessage) (engine.StepOutput, error) {
	cfg, err := parseMediaConfig(node)
	if err != nil {
		return engine.StepOutput{}, err
	}

	prompt := buildMediaPrompt(cfg.Prompt, inputs)

	provider, err := e.resolveProvider(ctx, cfg)
	if err != nil {
		return engine.StepOutput{}, err
	}

	req := MediaRequest{
		Model:       cfg.Model,
		Prompt:      prompt,
		AspectRatio: cfg.AspectRatio,
		Voice:       cfg.Voice,
		Duration:    cfg.Duration,
		NumOutputs:  cfg.NumOutputs,
	}

	jobID, err := provider.Submit(ctx, req)
	if err != nil {
		return engine.StepOutput{}, fmt.Errorf("media executor: submit: %w", err)
	}

	status, err := pollUntilDone(ctx, provider, jobID)
	if err != nil {
		return engine.StepOutput{}, fmt.Errorf("media executor: poll: %w", err)
	}

	if status.State == "failed" {
		return engine.StepOutput{}, fmt.Errorf("media executor: generation failed: %s", status.Error)
	}

	outputData, err := json.Marshal(status.Output)
	if err != nil {
		return engine.StepOutput{}, fmt.Errorf("media executor: marshal output: %w", err)
	}

	return engine.StepOutput{
		Data: json.RawMessage(outputData),
	}, nil
}

// ExecuteStream runs a media node with progress updates streamed as chunks.
func (e *MediaExecutor) ExecuteStream(ctx context.Context, node domain.NodeInstance, inputs map[string]json.RawMessage) (<-chan engine.ExecutorStreamChunk, <-chan engine.StepOutput, <-chan error) {
	chunks := make(chan engine.ExecutorStreamChunk, 64)
	result := make(chan engine.StepOutput, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunks)
		defer close(result)
		defer close(errCh)

		cfg, err := parseMediaConfig(node)
		if err != nil {
			errCh <- err
			return
		}

		prompt := buildMediaPrompt(cfg.Prompt, inputs)

		provider, err := e.resolveProvider(ctx, cfg)
		if err != nil {
			errCh <- err
			return
		}

		req := MediaRequest{
			Model:       cfg.Model,
			Prompt:      prompt,
			AspectRatio: cfg.AspectRatio,
			Voice:       cfg.Voice,
			Duration:    cfg.Duration,
			NumOutputs:  cfg.NumOutputs,
		}

		jobID, err := provider.Submit(ctx, req)
		if err != nil {
			errCh <- fmt.Errorf("media executor: submit: %w", err)
			return
		}

		chunks <- engine.ExecutorStreamChunk{Content: "Generating media..."}

		startTime := time.Now()
		deadline := startTime.Add(pollTimeout)
		interval := pollInitialInterval

		for {
			if time.Now().After(deadline) {
				errCh <- fmt.Errorf("media executor: poll timeout after %s", pollTimeout)
				return
			}

			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case <-time.After(interval):
			}

			status, err := provider.Poll(ctx, jobID)
			if err != nil {
				errCh <- fmt.Errorf("media executor: poll: %w", err)
				return
			}

			elapsed := time.Since(startTime).Truncate(time.Second)

			switch status.State {
			case "succeeded":
				outputData, marshalErr := json.Marshal(status.Output)
				if marshalErr != nil {
					errCh <- fmt.Errorf("media executor: marshal output: %w", marshalErr)
					return
				}
				chunks <- engine.ExecutorStreamChunk{Content: status.Output}
				result <- engine.StepOutput{
					Data: json.RawMessage(outputData),
				}
				return

			case "failed":
				errCh <- fmt.Errorf("media executor: generation failed: %s", status.Error)
				return

			case "cancelled":
				errCh <- fmt.Errorf("media executor: generation cancelled")
				return

			default:
				chunks <- engine.ExecutorStreamChunk{
					Content: fmt.Sprintf("Generating... (%s elapsed)", elapsed),
				}
			}

			// Exponential backoff capped at pollMaxInterval
			interval = time.Duration(float64(interval) * 1.5)
			if interval > pollMaxInterval {
				interval = pollMaxInterval
			}
		}
	}()

	return chunks, result, errCh
}

// resolveProvider determines the media provider from credential or registry.
func (e *MediaExecutor) resolveProvider(ctx context.Context, cfg domain.MediaNodeConfig) (Provider, error) {
	if cfg.CredentialID != "" {
		credID, parseErr := uuid.Parse(cfg.CredentialID)
		if parseErr != nil {
			return nil, fmt.Errorf("media executor: invalid credential_id: %w", parseErr)
		}
		tenantID := engine.TenantIDFromContext(ctx)
		cred, credErr := e.credentials.Get(ctx, credID, tenantID)
		if credErr != nil {
			return nil, fmt.Errorf("media executor: load credential: %w", credErr)
		}
		apiKey, decErr := e.credentials.GetDecrypted(ctx, credID, tenantID)
		if decErr != nil {
			return nil, fmt.Errorf("media executor: decrypt credential: %w", decErr)
		}
		return createProviderFromCredential(cred.Provider, apiKey)
	}

	providerName := cfg.Provider
	if providerName == "" {
		return nil, fmt.Errorf("media executor: no provider specified in node config")
	}

	provider, err := e.providers.Get(providerName)
	if err != nil {
		return nil, fmt.Errorf("media executor: %w", err)
	}
	return provider, nil
}

// createProviderFromCredential creates a media provider from a stored credential.
func createProviderFromCredential(providerName, apiKey string) (Provider, error) {
	switch providerName {
	case "replicate":
		return NewReplicate(apiKey), nil
	case "openai":
		return NewDALLE(apiKey), nil
	default:
		return nil, fmt.Errorf("unsupported media credential provider: %s", providerName)
	}
}

// parseMediaConfig unmarshals the node config into a MediaNodeConfig.
func parseMediaConfig(node domain.NodeInstance) (domain.MediaNodeConfig, error) {
	var cfg domain.MediaNodeConfig
	if err := json.Unmarshal(node.Config, &cfg); err != nil {
		return cfg, fmt.Errorf("media executor: unmarshal config: %w", err)
	}
	return cfg, nil
}

// buildMediaPrompt renders the prompt template with input data.
func buildMediaPrompt(template string, inputs map[string]json.RawMessage) string {
	inputText := concatenateInputs(inputs)

	if template == "" {
		return inputText
	}

	if strings.Contains(template, "{{input}}") {
		return strings.ReplaceAll(template, "{{input}}", inputText)
	}

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

// pollUntilDone polls a provider job with exponential backoff until completion or timeout.
func pollUntilDone(ctx context.Context, provider Provider, jobID string) (MediaStatus, error) {
	deadline := time.Now().Add(pollTimeout)
	interval := pollInitialInterval

	for {
		if time.Now().After(deadline) {
			return MediaStatus{}, fmt.Errorf("timeout after %s", pollTimeout)
		}

		select {
		case <-ctx.Done():
			return MediaStatus{}, ctx.Err()
		case <-time.After(interval):
		}

		status, err := provider.Poll(ctx, jobID)
		if err != nil {
			slog.Warn("media poll error, retrying", "job_id", jobID, "error", err)
			interval = time.Duration(float64(interval) * 1.5)
			if interval > pollMaxInterval {
				interval = pollMaxInterval
			}
			continue
		}

		switch status.State {
		case "succeeded", "failed", "cancelled":
			return status, nil
		}

		interval = time.Duration(float64(interval) * 1.5)
		if interval > pollMaxInterval {
			interval = pollMaxInterval
		}
	}
}
