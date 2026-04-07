import { useCallback, useEffect, useState } from 'react';
import type { AgentNodeConfig, TaskType, PortType, ProviderInfo, Credential } from '../../lib/types';
import { usePipelineStore } from '../../stores/pipelineStore';
import { api } from '../../lib/api';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const TASK_TYPES: TaskType[] = [
  'script',
  'image_prompt',
  'video_prompt',
  'audio_prompt',
  'character_prompt',
  'story',
  'prompt_enhancement',
  'story_to_prompt',
  'custom',
];

const PORT_TYPES: PortType[] = [
  'text',
  'image_prompt',
  'video_prompt',
  'audio_prompt',
  'image',
  'video',
  'audio',
  'json',
  'any',
];

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
  minHeight: 140,
  resize: 'vertical',
};

const warningBadgeStyle: React.CSSProperties = {
  display: 'inline-block',
  marginLeft: 6,
  padding: '1px 6px',
  borderRadius: 3,
  background: '#854d0e',
  color: '#fef08a',
  fontSize: 10,
  fontWeight: 600,
};

const warningBoxStyle: React.CSSProperties = {
  padding: '8px 10px',
  borderRadius: 4,
  background: '#422006',
  border: '1px solid #854d0e',
  color: '#fef08a',
  fontSize: 12,
  marginBottom: 12,
};

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface AgentInspectorProps {
  nodeId: string;
  label: string;
  config: AgentNodeConfig;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function AgentInspector({ nodeId, label, config }: AgentInspectorProps) {
  const updateNodeConfig = usePipelineStore((s) => s.updateNodeConfig);
  const updateNodeLabel = usePipelineStore((s) => s.updateNodeLabel);

  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [providersLoaded, setProvidersLoaded] = useState(false);
  const [credentials, setCredentials] = useState<Credential[]>([]);

  useEffect(() => {
    api.get<ProviderInfo[]>('/providers')
      .then((data) => {
        setProviders(Array.isArray(data) ? data : []);
        setProvidersLoaded(true);
      })
      .catch(() => {
        setProvidersLoaded(true);
      });

    api.credentials
      .list()
      .then((creds) => setCredentials(creds))
      .catch(() => {
        // credentials not available
      });
  }, []);

  const selectedProvider = providers.find((p) => p.name === config.provider);
  const modelsForProvider = selectedProvider?.models ?? [];

  const update = useCallback(
    (patch: Partial<AgentNodeConfig>) => {
      updateNodeConfig(nodeId, (prev) => ({ ...prev, ...patch }));
    },
    [nodeId, updateNodeConfig],
  );

  const updateRole = useCallback(
    (patch: Partial<AgentNodeConfig['role']>) => {
      updateNodeConfig(nodeId, (prev) => {
        const p = prev as AgentNodeConfig;
        return { ...p, role: { ...p.role, ...patch } };
      });
    },
    [nodeId, updateNodeConfig],
  );

  const updateTask = useCallback(
    (patch: Partial<AgentNodeConfig['task']>) => {
      updateNodeConfig(nodeId, (prev) => {
        const p = prev as AgentNodeConfig;
        return { ...p, task: { ...p.task, ...patch } };
      });
    },
    [nodeId, updateNodeConfig],
  );

  const handleProviderChange = useCallback(
    (providerName: string) => {
      const provider = providers.find((p) => p.name === providerName);
      const firstModel = provider?.models[0] ?? '';
      update({ provider: providerName, model: firstModel });
    },
    [providers, update],
  );

  const noProvidersAvailable =
    providersLoaded && providers.every((p) => !p.available);

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
        Agent Configuration
      </div>

      {noProvidersAvailable && (
        <div style={warningBoxStyle}>
          No LLM providers configured. Set GEMINI_API_KEY, OPENAI_API_KEY, or
          OPENROUTER_API_KEY.
        </div>
      )}

      <div style={fieldGroup}>
        <label style={labelStyle}>Label</label>
        <input
          style={inputStyle}
          value={label}
          onChange={(e) => updateNodeLabel(nodeId, e.target.value)}
        />
      </div>

      <div style={fieldGroup}>
        <label style={labelStyle}>Persona</label>
        <input
          style={inputStyle}
          value={config.role.persona}
          onChange={(e) => updateRole({ persona: e.target.value })}
        />
      </div>

      <div style={fieldGroup}>
        <label style={labelStyle}>System Prompt</label>
        <textarea
          style={textareaStyle}
          value={config.role.system_prompt}
          onChange={(e) => updateRole({ system_prompt: e.target.value })}
        />
      </div>

      <div style={fieldGroup}>
        <label style={labelStyle}>
          Provider
          {selectedProvider && !selectedProvider.available && (
            <span style={warningBadgeStyle}>No API Key</span>
          )}
        </label>
        <select
          style={inputStyle}
          value={config.provider}
          onChange={(e) => handleProviderChange(e.target.value)}
        >
          {providers.map((p) => (
            <option key={p.name} value={p.name}>
              {p.name}
            </option>
          ))}
        </select>
      </div>

      <div style={fieldGroup}>
        <label style={labelStyle}>Model</label>
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
          onChange={(e) => update({ temperature: parseFloat(e.target.value) })}
          style={{ width: '100%' }}
        />
      </div>

      <div style={fieldGroup}>
        <label style={labelStyle}>Max Tokens</label>
        <input
          type="number"
          style={inputStyle}
          value={config.max_tokens}
          min={1}
          onChange={(e) =>
            update({ max_tokens: parseInt(e.target.value, 10) || 1 })
          }
        />
      </div>

      <div style={fieldGroup}>
        <label style={labelStyle}>Task Type</label>
        <select
          style={inputStyle}
          value={config.task.task_type}
          onChange={(e) =>
            updateTask({ task_type: e.target.value as TaskType })
          }
        >
          {TASK_TYPES.map((t) => (
            <option key={t} value={t}>
              {t}
            </option>
          ))}
        </select>
      </div>

      <div style={fieldGroup}>
        <label style={labelStyle}>Output Type</label>
        <select
          style={inputStyle}
          value={config.task.output_type}
          onChange={(e) =>
            updateTask({ output_type: e.target.value as PortType })
          }
        >
          {PORT_TYPES.map((t) => (
            <option key={t} value={t}>
              {t}
            </option>
          ))}
        </select>
      </div>

      <div style={fieldGroup}>
        <label style={labelStyle}>Template</label>
        <textarea
          style={textareaStyle}
          value={config.task.template}
          onChange={(e) => updateTask({ template: e.target.value })}
          placeholder="Use {{input}} to reference incoming data"
        />
      </div>
    </div>
  );
}
