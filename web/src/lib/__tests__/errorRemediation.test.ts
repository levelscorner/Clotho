import { describe, it, expect } from 'vitest';
import { mapError } from '../errorRemediation';

describe('mapError', () => {
  it('returns "No error reported" for null input', () => {
    const result = mapError(null);
    expect(result.summary).toBe('No error reported');
  });

  it('returns "No error reported" for undefined input', () => {
    const result = mapError(undefined);
    expect(result.summary).toBe('No error reported');
  });

  it('returns "No error reported" for empty string', () => {
    const result = mapError('');
    expect(result.summary).toBe('No error reported');
  });

  it('matches rate limit errors', () => {
    const result = mapError('OpenAI returned: rate limit exceeded');
    expect(result.summary).toBe('Rate limit hit');
    expect(result.hint).toContain('30s');
  });

  it('matches API key errors (api key)', () => {
    const result = mapError('missing api key for provider');
    expect(result.summary).toBe('API key missing or invalid');
  });

  it('matches API key errors (unauthorized)', () => {
    const result = mapError('request failed: unauthorized');
    expect(result.summary).toBe('API key missing or invalid');
  });

  it('matches API key errors (401)', () => {
    const result = mapError('HTTP 401 from provider');
    expect(result.summary).toBe('API key missing or invalid');
  });

  it('matches timeout errors', () => {
    const result = mapError('request timeout after 60s');
    expect(result.summary).toBe('Provider timed out');
    expect(result.hint).toContain('Kokoro');
  });

  it('matches context deadline errors', () => {
    const result = mapError('context deadline exceeded');
    expect(result.summary).toBe('Provider timed out');
  });

  it('matches connection refused errors', () => {
    const result = mapError('dial tcp: connection refused');
    expect(result.summary).toBe("Local server isn't running");
    expect(result.hint).toContain('make dev-full');
  });

  it('matches ECONNREFUSED errors', () => {
    const result = mapError('fetch failed: ECONNREFUSED 127.0.0.1:8188');
    expect(result.summary).toBe("Local server isn't running");
  });

  it('matches not running errors', () => {
    const result = mapError('ComfyUI not running on :8188');
    expect(result.summary).toBe("Local server isn't running");
  });

  it('matches model not found errors', () => {
    const result = mapError('model not found: gpt-5');
    expect(result.summary).toBe('Model name mismatch');
  });

  it('matches 404 errors', () => {
    const result = mapError('HTTP 404: model llama3.5 not available');
    // "not found" pattern is "model not found|404" — 404 alone should match too
    expect(result.summary).toBe('Model name mismatch');
  });

  it('matches ollama no models pulled', () => {
    const result = mapError('ollama has no models pulled');
    expect(result.summary).toBe('Ollama has no models installed');
    expect(result.hint).toContain('ollama pull');
  });

  it('matches comfyui invalid workflow', () => {
    const result = mapError('invalid workflow: missing node');
    expect(result.summary).toBe('ComfyUI rejected the workflow');
  });

  it('matches validation failed errors', () => {
    const result = mapError('validation failed: checkpoint missing');
    expect(result.summary).toBe('ComfyUI rejected the workflow');
  });

  it('prefers more-specific "no models pulled" over generic 404', () => {
    const result = mapError('ollama: no models pulled (404)');
    expect(result.summary).toBe('Ollama has no models installed');
  });

  it('falls back for unknown errors', () => {
    const result = mapError('something completely unexpected happened');
    expect(result.summary).toBe('Step failed');
    expect(result.hint).toContain('make dev-logs');
  });

  it('is case-insensitive for all patterns', () => {
    expect(mapError('RATE LIMIT').summary).toBe('Rate limit hit');
    expect(mapError('Unauthorized').summary).toBe('API key missing or invalid');
    expect(mapError('TIMEOUT').summary).toBe('Provider timed out');
  });
});
