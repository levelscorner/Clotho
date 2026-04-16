import { useCallback } from 'react';
import type { AgentNodeConfig } from '../../../lib/types';
import { InspectorGroup } from '../InspectorGroup';
import { capabilitiesFor } from '../../../lib/llmCapabilities';
import { fieldGroup, inputStyle, labelStyle, helperTextStyle } from './sectionStyles';

interface SamplingSectionProps {
  config: AgentNodeConfig;
  onChange: (patch: Partial<AgentNodeConfig>) => void;
}

/**
 * Sampling knobs. Universal fields (temperature, max_tokens, top_p,
 * stop_sequences) always render. Near-universal fields (top_k, seed,
 * frequency_penalty, presence_penalty) are gated by
 * web/src/lib/llmCapabilities.ts. A knob the provider doesn't honor
 * is hidden entirely rather than greyed out — the mental model is
 * "the inspector only shows knobs that matter for the current model".
 *
 * All pointer-style fields default to `undefined` ("use provider
 * default"). Clearing an input sets the field back to undefined so the
 * outbound request payload stays minimal.
 */
export function SamplingSection({ config, onChange }: SamplingSectionProps) {
  const caps = capabilitiesFor(config.provider);

  const setNumber = useCallback(
    (key: keyof AgentNodeConfig, raw: string) => {
      if (raw === '') {
        onChange({ [key]: undefined } as Partial<AgentNodeConfig>);
        return;
      }
      const value = parseFloat(raw);
      if (Number.isFinite(value)) {
        onChange({ [key]: value } as Partial<AgentNodeConfig>);
      }
    },
    [onChange],
  );

  const setInt = useCallback(
    (key: keyof AgentNodeConfig, raw: string) => {
      if (raw === '') {
        onChange({ [key]: undefined } as Partial<AgentNodeConfig>);
        return;
      }
      const value = parseInt(raw, 10);
      if (Number.isFinite(value)) {
        onChange({ [key]: value } as Partial<AgentNodeConfig>);
      }
    },
    [onChange],
  );

  const setStopSequences = useCallback(
    (raw: string) => {
      const list = raw
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      onChange({ stop_sequences: list.length > 0 ? list : undefined });
    },
    [onChange],
  );

  return (
    <InspectorGroup title="Sampling">
      <div style={fieldGroup}>
        <label style={labelStyle}>
          Temperature: {config.temperature.toFixed(1)}
        </label>
        <input
          type="range"
          min={0}
          max={2}
          step={0.1}
          value={config.temperature}
          onChange={(e) => onChange({ temperature: parseFloat(e.target.value) })}
          style={{ width: '100%' }}
        />
      </div>

      <div style={fieldGroup}>
        <label style={labelStyle}>Max Tokens</label>
        <input
          type="number"
          style={inputStyle}
          min={1}
          value={config.max_tokens}
          onChange={(e) =>
            onChange({ max_tokens: parseInt(e.target.value, 10) || 1 })
          }
        />
      </div>

      <div style={fieldGroup}>
        <label style={labelStyle}>Top-P</label>
        <input
          type="number"
          style={inputStyle}
          min={0}
          max={1}
          step={0.05}
          placeholder="default"
          value={config.top_p ?? ''}
          onChange={(e) => setNumber('top_p', e.target.value)}
        />
        <div style={helperTextStyle}>
          Nucleus sampling. Lower values make output more focused. Leave
          blank to use the provider default.
        </div>
      </div>

      {caps.topK && (
        <div style={fieldGroup}>
          <label style={labelStyle}>Top-K</label>
          <input
            type="number"
            style={inputStyle}
            min={1}
            placeholder="default"
            value={config.top_k ?? ''}
            onChange={(e) => setInt('top_k', e.target.value)}
          />
        </div>
      )}

      {caps.stopSequences && (
        <div style={fieldGroup}>
          <label style={labelStyle}>Stop Sequences</label>
          <input
            style={inputStyle}
            placeholder="comma-separated, e.g. ###,END"
            value={(config.stop_sequences ?? []).join(', ')}
            onChange={(e) => setStopSequences(e.target.value)}
          />
          <div style={helperTextStyle}>
            Model stops generating when it produces any of these strings.
          </div>
        </div>
      )}

      {caps.seed && (
        <div style={fieldGroup}>
          <label style={labelStyle}>Seed</label>
          <input
            type="number"
            style={inputStyle}
            placeholder="random"
            value={config.seed ?? ''}
            onChange={(e) => setInt('seed', e.target.value)}
          />
          <div style={helperTextStyle}>
            Same seed + same prompt = reproducible output (best-effort).
          </div>
        </div>
      )}

      {caps.frequencyPenalty && (
        <div style={fieldGroup}>
          <label style={labelStyle}>Frequency Penalty</label>
          <input
            type="number"
            style={inputStyle}
            min={-2}
            max={2}
            step={0.1}
            placeholder="default"
            value={config.frequency_penalty ?? ''}
            onChange={(e) => setNumber('frequency_penalty', e.target.value)}
          />
        </div>
      )}

      {caps.presencePenalty && (
        <div style={fieldGroup}>
          <label style={labelStyle}>Presence Penalty</label>
          <input
            type="number"
            style={inputStyle}
            min={-2}
            max={2}
            step={0.1}
            placeholder="default"
            value={config.presence_penalty ?? ''}
            onChange={(e) => setNumber('presence_penalty', e.target.value)}
          />
        </div>
      )}
    </InspectorGroup>
  );
}
