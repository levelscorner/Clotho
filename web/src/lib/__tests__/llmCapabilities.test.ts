import { describe, it, expect } from 'vitest';
import { CAPABILITIES, capabilitiesFor, type ProviderName } from '../llmCapabilities';

describe('CAPABILITIES', () => {
  const providers: ProviderName[] = ['openai', 'gemini', 'openrouter', 'ollama'];
  const flagKeys = [
    'topK',
    'seed',
    'frequencyPenalty',
    'presencePenalty',
    'stopSequences',
    'jsonMode',
    'jsonSchema',
    'tools',
  ] as const;

  it.each(providers)('declares every capability flag for %s', (provider) => {
    const caps = CAPABILITIES[provider];
    for (const key of flagKeys) {
      expect(typeof caps[key], `${provider}.${key}`).toBe('boolean');
    }
    expect(typeof caps.vision).toBe('function');
    expect(typeof caps.reasoning).toBe('function');
  });

  it('mirrors the backend Go table: openai has no topK', () => {
    expect(CAPABILITIES.openai.topK).toBe(false);
  });

  it('mirrors the backend Go table: ollama has no frequency/presence penalty', () => {
    expect(CAPABILITIES.ollama.frequencyPenalty).toBe(false);
    expect(CAPABILITIES.ollama.presencePenalty).toBe(false);
  });

  it('mirrors the backend Go table: every provider supports tools + jsonMode + jsonSchema', () => {
    for (const provider of providers) {
      expect(CAPABILITIES[provider].tools, provider).toBe(true);
      expect(CAPABILITIES[provider].jsonMode, provider).toBe(true);
      expect(CAPABILITIES[provider].jsonSchema, provider).toBe(true);
    }
  });

  it('detects vision models per provider', () => {
    expect(CAPABILITIES.openai.vision('gpt-4o')).toBe(true);
    expect(CAPABILITIES.openai.vision('gpt-3.5-turbo')).toBe(false);
    expect(CAPABILITIES.ollama.vision('llava')).toBe(true);
    expect(CAPABILITIES.ollama.vision('llama3.1')).toBe(false);
    expect(CAPABILITIES.gemini.vision('gemini-2.5-pro')).toBe(true);
  });

  it('detects reasoning models per provider', () => {
    expect(CAPABILITIES.openai.reasoning('o3-mini')).toBe('effort');
    expect(CAPABILITIES.openai.reasoning('o1-preview')).toBe('effort');
    expect(CAPABILITIES.openai.reasoning('gpt-4o')).toBe(null);
    expect(CAPABILITIES.gemini.reasoning('gemini-2.5-pro')).toBe('budget');
    expect(CAPABILITIES.gemini.reasoning('gemini-1.5-pro')).toBe(null);
    expect(CAPABILITIES.ollama.reasoning('gemma4:26b')).toBe('stream');
    expect(CAPABILITIES.ollama.reasoning('llama3.1')).toBe(null);
  });
});

describe('capabilitiesFor', () => {
  it('returns caps for known providers', () => {
    expect(capabilitiesFor('openai').seed).toBe(true);
  });

  it('returns safe zero-value for unknown providers', () => {
    const caps = capabilitiesFor('anthropic');
    expect(caps.topK).toBe(false);
    expect(caps.seed).toBe(false);
    expect(caps.tools).toBe(false);
    expect(caps.jsonMode).toBe(false);
    expect(caps.vision('claude-3-opus')).toBe(false);
    expect(caps.reasoning('claude-3-opus')).toBe(null);
  });
});
