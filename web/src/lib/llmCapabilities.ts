// Frontend mirror of internal/llm/capabilities.go. The Prompt Component
// inspector reads this table to decide which knobs to render for the
// currently selected provider. Keep in sync with the backend — a test in
// __tests__/llmCapabilities.test.ts locks every provider × capability cell.

export type ProviderName = 'openai' | 'gemini' | 'openrouter' | 'ollama';

export interface ProviderCapabilities {
  topK: boolean;
  seed: boolean;
  frequencyPenalty: boolean;
  presencePenalty: boolean;
  stopSequences: boolean;
  jsonMode: boolean;
  jsonSchema: boolean;
  tools: boolean;
  /** Returns true when the specific model supports image inputs. */
  vision: (model: string) => boolean;
  /**
   * Returns the reasoning-control style a model supports:
   *   'effort' — OpenAI o-series `reasoning_effort` enum
   *   'budget' — Gemini 2.5 `thinkingConfig.thinkingBudget`
   *   'stream' — Ollama reasoning variants (no request knob, just delta.reasoning)
   *   null     — not a reasoning model
   */
  reasoning: (model: string) => 'effort' | 'budget' | 'stream' | null;
}

const VISION_MODELS_OPENAI = new Set([
  'gpt-4o',
  'gpt-4o-mini',
  'gpt-4-turbo',
  'gpt-4-vision-preview',
]);

const VISION_MODELS_OLLAMA = new Set([
  'llava',
  'llava:7b',
  'llava:13b',
  'llava:34b',
  'bakllava',
  'llama3.2-vision',
  'llama3.2-vision:11b',
  'llama3.2-vision:90b',
  'minicpm-v',
]);

const REASONING_OLLAMA = new Set([
  'gemma4:26b',
  'qwen3:8b',
  'qwen3:14b',
  'qwen3:32b',
  'deepseek-r1',
]);

export const CAPABILITIES: Record<ProviderName, ProviderCapabilities> = {
  openai: {
    topK: false,
    seed: true,
    frequencyPenalty: true,
    presencePenalty: true,
    stopSequences: true,
    jsonMode: true,
    jsonSchema: true,
    tools: true,
    vision: (model) => VISION_MODELS_OPENAI.has(model),
    reasoning: (model) => (/^o[134]/.test(model) ? 'effort' : null),
  },
  gemini: {
    topK: true,
    seed: true,
    frequencyPenalty: true,
    presencePenalty: true,
    stopSequences: true,
    jsonMode: true,
    jsonSchema: true,
    tools: true,
    vision: () => true,
    reasoning: (model) => (model.includes('2.5') ? 'budget' : null),
  },
  openrouter: {
    topK: true,
    seed: true,
    frequencyPenalty: true,
    presencePenalty: true,
    stopSequences: true,
    jsonMode: true,
    jsonSchema: true,
    tools: true,
    vision: () => false,
    reasoning: () => null,
  },
  ollama: {
    topK: true,
    seed: true,
    frequencyPenalty: false,
    presencePenalty: false,
    stopSequences: true,
    jsonMode: true,
    jsonSchema: true,
    tools: true,
    vision: (model) => VISION_MODELS_OLLAMA.has(model),
    reasoning: (model) => (REASONING_OLLAMA.has(model) ? 'stream' : null),
  },
};

export function capabilitiesFor(provider: string): ProviderCapabilities {
  if (provider in CAPABILITIES) {
    return CAPABILITIES[provider as ProviderName];
  }
  return {
    topK: false,
    seed: false,
    frequencyPenalty: false,
    presencePenalty: false,
    stopSequences: false,
    jsonMode: false,
    jsonSchema: false,
    tools: false,
    vision: () => false,
    reasoning: () => null,
  };
}
