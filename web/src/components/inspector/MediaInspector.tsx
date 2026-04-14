import { useCallback, useEffect, useState } from 'react';
import type { MediaNodeConfig, Credential, StepResult } from '../../lib/types';
import { usePipelineStore } from '../../stores/pipelineStore';
import { api } from '../../lib/api';
import { InspectorGroup } from './InspectorGroup';
import { OllamaModelDropdown } from './OllamaModelDropdown';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const PROVIDERS = [
  'replicate',
  'openai',
  'elevenlabs',
  'kokoro', // local TTS via Kokoro-FastAPI (http://localhost:8880)
  'comfyui', // local image gen via ComfyUI (http://localhost:8188)
] as const;

const MODELS_BY_PROVIDER: Record<string, string[]> = {
  replicate: ['flux-1.1-pro', 'sdxl', 'stable-video-diffusion', 'musicgen'],
  openai: ['dall-e-3', 'dall-e-2', 'tts-1', 'tts-1-hd'],
  elevenlabs: ['eleven_multilingual_v2', 'eleven_turbo_v2'],
  kokoro: ['kokoro'], // single model; the real knob for Kokoro is voice
  comfyui: ['flux1-schnell'], // single checkpoint for now; all-in-one fp8
};

// Providers that run entirely on the local machine. Nodes using these show
// "local" in place of a dollar cost and get a tinted cost badge.
export const LOCAL_MEDIA_PROVIDERS = new Set<string>(['kokoro', 'comfyui']);

const ASPECT_RATIOS = ['1:1', '16:9', '9:16', '4:3'] as const;

// Voice sets per audio provider. Kokoro's v1_0 voices are exposed as-is.
// The KOKORO_VOICES prefix encodes language+gender: `af_` American female,
// `am_` American male, `bf_` British female, `bm_` British male.
const OPENAI_VOICES = ['alloy', 'echo', 'fable', 'onyx', 'nova', 'shimmer'] as const;
const KOKORO_VOICES = [
  'af_bella',
  'af_sarah',
  'af_nicole',
  'af_sky',
  'am_adam',
  'am_michael',
  'bf_emma',
  'bf_isabella',
  'bm_george',
  'bm_lewis',
] as const;
const VOICES_BY_PROVIDER: Record<string, readonly string[]> = {
  openai: OPENAI_VOICES,
  elevenlabs: OPENAI_VOICES, // ElevenLabs uses voice IDs; placeholder set kept in sync with OpenAI for now.
  kokoro: KOKORO_VOICES,
  replicate: OPENAI_VOICES,
};

// Back-compat alias: old code referenced a single VOICES constant.
const VOICES = OPENAI_VOICES;

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

const fieldGroup: React.CSSProperties = {
  marginBottom: 12,
};

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: 11,
  fontWeight: 600,
  color: '#64748b',
  marginBottom: 4,
  textTransform: 'uppercase',
  letterSpacing: '0.04em',
};

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '6px 8px',
  borderRadius: 4,
  border: '1px solid #334155',
  background: '#1a1c2e',
  color: '#e2e8f0',
  fontSize: 13,
};

const textareaStyle: React.CSSProperties = {
  ...inputStyle,
  minHeight: 100,
  resize: 'vertical',
};

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface MediaInspectorProps {
  nodeId: string;
  label: string;
  config: MediaNodeConfig;
  stepResult?: StepResult;
}

function errorImplicatesAdvanced(step?: StepResult): boolean {
  if (!step || step.status !== 'failed' || !step.error) return false;
  const e = step.error.toLowerCase();
  return (
    e.includes('model') ||
    e.includes('credential') ||
    e.includes('api key') ||
    e.includes('api_key')
  );
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function MediaInspector({ nodeId, label, config, stepResult }: MediaInspectorProps) {
  const updateNodeConfig = usePipelineStore((s) => s.updateNodeConfig);
  const updateNodeLabel = usePipelineStore((s) => s.updateNodeLabel);

  const [credentials, setCredentials] = useState<Credential[]>([]);

  useEffect(() => {
    api.credentials
      .list()
      .then((creds) => setCredentials(creds))
      .catch(() => {
        // credentials not available
      });
  }, []);

  const update = useCallback(
    (patch: Partial<MediaNodeConfig>) => {
      updateNodeConfig(nodeId, (prev) => ({ ...prev, ...patch }));
    },
    [nodeId, updateNodeConfig],
  );

  const handleProviderChange = useCallback(
    (provider: string) => {
      const models = MODELS_BY_PROVIDER[provider] ?? [];
      update({ provider, model: models[0] ?? '' });
    },
    [update],
  );

  const modelsForProvider = MODELS_BY_PROVIDER[config.provider] ?? [];
  const showAspectRatio = config.media_type === 'image' || config.media_type === 'video';
  const showVoice = config.media_type === 'audio';
  const showNumOutputs = config.media_type === 'image';

  return (
    <div>
      <div
        style={{
          fontSize: 12,
          fontWeight: 700,
          color: '#94a3b8',
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          marginBottom: 14,
          paddingBottom: 8,
          borderBottom: '1px solid #1e2030',
        }}
      >
        Media Configuration
      </div>

      <InspectorGroup title="Basics" defaultOpen>
        <div style={fieldGroup}>
          <label style={labelStyle}>Label</label>
          <input
            style={inputStyle}
            value={label}
            onChange={(e) => updateNodeLabel(nodeId, e.target.value)}
          />
        </div>

        <div style={fieldGroup}>
          <label style={labelStyle}>Provider</label>
          <select
            style={inputStyle}
            value={config.provider}
            onChange={(e) => handleProviderChange(e.target.value)}
          >
            {PROVIDERS.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
        </div>

        <div style={fieldGroup}>
          <label style={labelStyle}>Model</label>
          {config.provider === 'ollama' ? (
            <OllamaModelDropdown
              value={config.model}
              onChange={(m) => update({ model: m })}
            />
          ) : (
            <select
              style={inputStyle}
              value={config.model}
              onChange={(e) => update({ model: e.target.value })}
            >
              {modelsForProvider.map((m) => (
                <option key={m} value={m}>
                  {m}
                </option>
              ))}
            </select>
          )}
        </div>

        <div style={fieldGroup}>
          <label style={labelStyle}>Prompt Template</label>
          <textarea
            style={textareaStyle}
            value={config.prompt}
            onChange={(e) => update({ prompt: e.target.value })}
            placeholder="Use {{input}} to reference incoming data"
          />
        </div>
      </InspectorGroup>

      <InspectorGroup
        title="Advanced"
        forceOpen={errorImplicatesAdvanced(stepResult)}
      >
        {showAspectRatio && (
        <div style={fieldGroup}>
          <label style={labelStyle}>Aspect Ratio</label>
          <select
            style={inputStyle}
            value={config.aspect_ratio ?? '1:1'}
            onChange={(e) => update({ aspect_ratio: e.target.value })}
          >
            {ASPECT_RATIOS.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </select>
        </div>
      )}

      {showVoice && (
        <div style={fieldGroup}>
          <label style={labelStyle}>Voice</label>
          <select
            style={inputStyle}
            value={config.voice ?? (VOICES_BY_PROVIDER[config.provider]?.[0] ?? 'alloy')}
            onChange={(e) => update({ voice: e.target.value })}
          >
            {(VOICES_BY_PROVIDER[config.provider] ?? VOICES).map((v) => (
              <option key={v} value={v}>
                {v}
              </option>
            ))}
          </select>
        </div>
      )}

      {showNumOutputs && (
        <div style={fieldGroup}>
          <label style={labelStyle}>Number of Outputs</label>
          <select
            style={inputStyle}
            value={config.num_outputs ?? 1}
            onChange={(e) => update({ num_outputs: parseInt(e.target.value, 10) })}
          >
            {[1, 2, 3, 4].map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
        </div>
      )}

      <div style={fieldGroup}>
        <label style={labelStyle}>Cost Cap ($)</label>
        <input
          type="number"
          style={inputStyle}
          value={config.cost_cap ?? ''}
          min={0}
          step={0.01}
          placeholder="No limit"
          onChange={(e) =>
            update({
              cost_cap: e.target.value ? parseFloat(e.target.value) : undefined,
            })
          }
        />
      </div>

      <div style={fieldGroup}>
        <label style={labelStyle}>API Key</label>
        {credentials.length > 0 ? (
          <select
            style={inputStyle}
            value={config.credential_id ?? ''}
            onChange={(e) =>
              update({
                credential_id: e.target.value || undefined,
              })
            }
          >
            <option value="">Use server default</option>
            {credentials.map((c) => (
              <option key={c.id} value={c.id}>
                {c.provider} — {c.label}
              </option>
            ))}
          </select>
        ) : (
          <div
            style={{
              fontSize: 12,
              color: '#55556a',
              padding: '6px 0',
            }}
          >
            No API keys saved.{' '}
            <span
              style={{
                color: '#e5a84b',
                cursor: 'pointer',
              }}
            >
              Add one in Settings
            </span>
          </div>
        )}
      </div>
      </InspectorGroup>
    </div>
  );
}
