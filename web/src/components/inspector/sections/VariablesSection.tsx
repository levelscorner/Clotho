import { useCallback, useState } from 'react';
import type { AgentNodeConfig } from '../../../lib/types';
import { InspectorGroup } from '../InspectorGroup';
import { fieldGroup, inputStyle, labelStyle, helperTextStyle } from './sectionStyles';

interface VariablesSectionProps {
  variables: Record<string, string> | undefined;
  onChange: (patch: Partial<AgentNodeConfig['role']>) => void;
}

/**
 * Variables section — name/value pairs substituted into both the system
 * prompt and the user template via `{{name}}` tokens. Undefined names
 * stay literal so `{{input}}` can still late-bind upstream data.
 *
 * Backed by cfg.Role.Variables on the backend (string→string map). Values
 * only; no typing — expansions are pure text. Empty names are rejected.
 */
export function VariablesSection({ variables, onChange }: VariablesSectionProps) {
  const entries = Object.entries(variables ?? {});
  const [draftName, setDraftName] = useState('');
  const [draftValue, setDraftValue] = useState('');

  const applyAll = useCallback(
    (next: Record<string, string>) => {
      onChange({ variables: next });
    },
    [onChange],
  );

  const renameKey = useCallback(
    (oldKey: string, newKey: string) => {
      const next: Record<string, string> = {};
      for (const [k, v] of entries) {
        if (k === oldKey) {
          if (newKey) next[newKey] = v;
        } else {
          next[k] = v;
        }
      }
      applyAll(next);
    },
    [entries, applyAll],
  );

  const updateValue = useCallback(
    (key: string, value: string) => {
      const next: Record<string, string> = {};
      for (const [k, v] of entries) {
        next[k] = k === key ? value : v;
      }
      applyAll(next);
    },
    [entries, applyAll],
  );

  const removeKey = useCallback(
    (key: string) => {
      const next: Record<string, string> = {};
      for (const [k, v] of entries) {
        if (k !== key) next[k] = v;
      }
      applyAll(next);
    },
    [entries, applyAll],
  );

  const addDraft = useCallback(() => {
    const name = draftName.trim();
    if (!name) return;
    const next = Object.fromEntries(entries);
    next[name] = draftValue;
    applyAll(next);
    setDraftName('');
    setDraftValue('');
  }, [draftName, draftValue, entries, applyAll]);

  return (
    <InspectorGroup title="Variables">
      <div style={fieldGroup}>
        <label style={labelStyle}>Defined variables</label>
        <div style={helperTextStyle}>
          Use <code>{'{{name}}'}</code> in the prompt and system prompt.
          Unknown names stay literal — <code>{'{{input}}'}</code> still
          refers to upstream step data.
        </div>
      </div>

      {entries.length === 0 && (
        <div style={{ ...helperTextStyle, marginBottom: 10 }}>
          No variables yet. Add one below.
        </div>
      )}

      {entries.map(([name, value]) => (
        <div
          key={name}
          style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr auto',
            gap: 6,
            marginBottom: 6,
          }}
        >
          <input
            aria-label={`Variable name (currently ${name})`}
            style={inputStyle}
            defaultValue={name}
            onBlur={(e) => {
              const next = e.target.value.trim();
              if (next && next !== name) renameKey(name, next);
            }}
          />
          <input
            aria-label={`Value for ${name}`}
            style={inputStyle}
            value={value}
            onChange={(e) => updateValue(name, e.target.value)}
          />
          <button
            type="button"
            aria-label={`Remove ${name}`}
            onClick={() => removeKey(name)}
            style={{
              background: 'transparent',
              border: '1px solid var(--surface-border)',
              color: 'var(--text-secondary)',
              borderRadius: 'var(--radius-sm)',
              padding: '0 10px',
              cursor: 'pointer',
            }}
          >
            {'\u2715'}
          </button>
        </div>
      ))}

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr auto',
          gap: 6,
          marginTop: 10,
          paddingTop: 10,
          borderTop: '1px dashed var(--surface-border)',
        }}
      >
        <input
          placeholder="name"
          style={inputStyle}
          value={draftName}
          onChange={(e) => setDraftName(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault();
              addDraft();
            }
          }}
        />
        <input
          placeholder="value"
          style={inputStyle}
          value={draftValue}
          onChange={(e) => setDraftValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault();
              addDraft();
            }
          }}
        />
        <button
          type="button"
          onClick={addDraft}
          disabled={!draftName.trim()}
          style={{
            background: 'var(--accent-soft)',
            border: '1px solid var(--surface-border)',
            color: 'var(--text-primary)',
            borderRadius: 'var(--radius-sm)',
            padding: '0 12px',
            cursor: draftName.trim() ? 'pointer' : 'not-allowed',
            opacity: draftName.trim() ? 1 : 0.5,
          }}
        >
          Add
        </button>
      </div>
    </InspectorGroup>
  );
}
