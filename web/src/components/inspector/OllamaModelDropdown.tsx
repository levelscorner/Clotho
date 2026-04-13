import { useEffect, useState } from 'react';
import { api } from '../../lib/api';
import type { OllamaModelsResponse } from '../../lib/api';
import './OllamaModelDropdown.css';

export interface OllamaModelDropdownProps {
  value: string;
  onChange: (modelName: string) => void;
  disabled?: boolean;
}

export type FetchState =
  | { kind: 'loading' }
  | { kind: 'ok'; models: string[] }
  | { kind: 'empty' }
  | { kind: 'not_running' }
  | { kind: 'error' };

/** Pure reducer: map an API response (or error) to a FetchState. */
export function deriveState(
  resp: OllamaModelsResponse | null,
  error: boolean = false,
): FetchState {
  if (error || resp === null) return { kind: 'error' };
  if (resp.status === 'ollama_not_running') return { kind: 'not_running' };
  const names = (resp.models ?? []).map((m) => m.name);
  if (names.length === 0) return { kind: 'empty' };
  return { kind: 'ok', models: names };
}

/**
 * Native <select> listing local Ollama models.
 * Handles loading, empty, and Ollama-not-running states with inline hints.
 */
export function OllamaModelDropdown({
  value,
  onChange,
  disabled = false,
}: OllamaModelDropdownProps) {
  const [state, setState] = useState<FetchState>({ kind: 'loading' });

  useEffect(() => {
    let cancelled = false;

    setState({ kind: 'loading' });
    api
      .fetchOllamaModels()
      .then((resp: OllamaModelsResponse) => {
        if (cancelled) return;
        setState(deriveState(resp));
      })
      .catch(() => {
        if (cancelled) return;
        setState({ kind: 'error' });
      });

    return () => {
      cancelled = true;
    };
  }, []);

  return renderState(state, value, onChange, disabled);
}

/** Pure view function — renders a FetchState. Exported for testability. */
export function renderState(
  state: FetchState,
  value: string,
  onChange: (v: string) => void,
  disabled: boolean,
): JSX.Element {
  if (state.kind === 'loading') {
    return (
      <div className="clotho-ollama-dropdown">
        <select
          className="clotho-ollama-dropdown__select"
          disabled
          aria-label="Ollama model"
        >
          <option>Loading models…</option>
        </select>
      </div>
    );
  }

  if (state.kind === 'ok') {
    const options = state.models.includes(value)
      ? state.models
      : value
        ? [value, ...state.models]
        : state.models;
    return (
      <div className="clotho-ollama-dropdown">
        <select
          className="clotho-ollama-dropdown__select"
          value={value}
          disabled={disabled}
          onChange={(e) => onChange(e.target.value)}
          aria-label="Ollama model"
        >
          {options.map((name) => (
            <option key={name} value={name}>
              {name}
            </option>
          ))}
        </select>
      </div>
    );
  }

  // empty / not_running / error — disabled select + inline hint
  const hint =
    state.kind === 'empty' ? (
      <>
        No models pulled. <code>ollama pull llama3.1</code> to start.
      </>
    ) : (
      <>
        Ollama not detected. <code>brew install ollama &amp;&amp; ollama serve</code> to enable.
      </>
    );

  return (
    <div className="clotho-ollama-dropdown">
      <select
        className="clotho-ollama-dropdown__select"
        disabled
        aria-label="Ollama model"
      >
        <option>No models available</option>
      </select>
      <div className="clotho-ollama-dropdown__hint">{hint}</div>
    </div>
  );
}
