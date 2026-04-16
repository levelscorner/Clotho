import { useCallback, useEffect, useState } from 'react';
import type {
  AgentNodeConfig,
  TaskType,
  PortType,
  ProviderInfo,
  Credential,
  StepResult,
} from '../../lib/types';
import { usePipelineStore } from '../../stores/pipelineStore';
import { api } from '../../lib/api';
import { InspectorGroup } from './InspectorGroup';
import { AboutNodeSection } from './AboutNodeSection';
import { describeNode } from '../../lib/nodeDescriptions';
import { OllamaModelDropdown } from './OllamaModelDropdown';
import { VariablesSection } from './sections/VariablesSection';
import { SamplingSection } from './sections/SamplingSection';
import { TestStepButton } from './TestStepButton';
import { fieldGroup, inputStyle, labelStyle, textareaStyle } from './sections/sectionStyles';

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
  stepResult?: StepResult;
}

/**
 * Heuristic: does the step error plausibly relate to the Credentials group
 * (model name, credential/API key)? Used to auto-expand it when an execution
 * fails for a reason the user can fix there.
 */
function errorImplicatesCredentials(step?: StepResult): boolean {
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

export function AgentInspector({ nodeId, label, config, stepResult }: AgentInspectorProps) {
  const updateNodeConfig = usePipelineStore((s) => s.updateNodeConfig);
  const updateNodeLabel = usePipelineStore((s) => s.updateNodeLabel);

  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [providersLoaded, setProvidersLoaded] = useState(false);
  const [credentials, setCredentials] = useState<Credential[]>([]);

  useEffect(() => {
    api
      .get<ProviderInfo[]>('/providers')
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
          color: 'var(--text-secondary)',
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          marginBottom: 14,
          paddingBottom: 8,
          borderBottom: '1px solid var(--surface-border)',
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

      <AboutNodeSection description={describeNode({ nodeType: 'agent' })} />

      {/* Test step in isolation — fast feedback loop without saving. */}
      <TestStepButton nodeId={nodeId} />

      {/* Notes — free-form annotation. Lives at the top so creators can
          jot down "breaks above 4k tokens" right next to the About text. */}
      <InspectorGroup title="Notes">
        <textarea
          aria-label="Node notes"
          placeholder="Free-form notes for your future self. Engine never reads this."
          value={config.notes ?? ''}
          onChange={(e) => update({ notes: e.target.value })}
          style={{
            ...textareaStyle,
            minHeight: 64,
            fontSize: 12,
            lineHeight: 1.45,
          }}
        />
      </InspectorGroup>

      {/* Basics — label, prompt, provider, model, system prompt, persona */}
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
          <label style={labelStyle}>Prompt</label>
          <textarea
            style={textareaStyle}
            value={config.task.template}
            onChange={(e) => updateTask({ template: e.target.value })}
            placeholder="Use {{input}} to reference incoming data"
          />
          <div
            style={{
              marginTop: 4,
              fontFamily:
                'var(--font-mono, "JetBrains Mono", ui-monospace, monospace)',
              fontSize: 10,
              color: 'var(--text-muted)',
            }}
          >
            Use {'{{input}}'} for upstream data. Add named vars below.
          </div>
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
      </InspectorGroup>

      {/* Variables — {{name}} substitution across prompts */}
      <VariablesSection
        variables={config.role.variables}
        onChange={updateRole}
      />

      {/* Sampling — temperature + top-p/top-k/seed/penalties, gated */}
      <SamplingSection config={config} onChange={update} />

      {/* Task routing — kept in its own collapsed group since users
          rarely touch task_type/output_type after initial setup */}
      <InspectorGroup title="Task Routing">
        <div style={fieldGroup}>
          <label style={labelStyle}>Task Type</label>
          <input
            list="clotho-task-types"
            style={inputStyle}
            value={config.task.task_type}
            onChange={(e) =>
              updateTask({ task_type: e.target.value as TaskType })
            }
            placeholder="Type or select..."
          />
          <datalist id="clotho-task-types">
            {TASK_TYPES.map((t) => (
              <option key={t} value={t} />
            ))}
          </datalist>
        </div>

        <div style={fieldGroup}>
          <label style={labelStyle}>Output Type</label>
          <input
            list="clotho-output-types"
            style={inputStyle}
            value={config.task.output_type}
            onChange={(e) =>
              updateTask({ output_type: e.target.value as PortType })
            }
            placeholder="Type or select..."
          />
          <datalist id="clotho-output-types">
            {PORT_TYPES.map((t) => (
              <option key={t} value={t} />
            ))}
          </datalist>
        </div>
      </InspectorGroup>

      {/* Credentials & Cost — API key selection + cost cap */}
      <InspectorGroup
        title="Credentials & Cost"
        forceOpen={errorImplicatesCredentials(stepResult)}
      >
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
                color: 'var(--text-muted)',
                padding: '6px 0',
              }}
            >
              No API keys saved.{' '}
              <span
                style={{
                  color: 'var(--accent)',
                  cursor: 'pointer',
                }}
              >
                Add one in Settings
              </span>
            </div>
          )}
        </div>

        <div style={fieldGroup}>
          <label style={labelStyle}>Cost Cap (USD)</label>
          <input
            type="number"
            style={inputStyle}
            min={0}
            step={0.01}
            placeholder="no cap"
            value={config.cost_cap ?? ''}
            onChange={(e) => {
              const raw = e.target.value;
              if (raw === '') {
                update({ cost_cap: undefined });
                return;
              }
              const n = parseFloat(raw);
              if (Number.isFinite(n)) update({ cost_cap: n });
            }}
          />
        </div>
      </InspectorGroup>
    </div>
  );
}
