import { describe, it, expect, vi } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { deriveState, renderState } from '../OllamaModelDropdown';

// Mock the CSS import to keep Vitest happy in the node environment.
vi.mock('../OllamaModelDropdown.css', () => ({}));

describe('OllamaModelDropdown — deriveState', () => {
  it('maps status=ok + models to { kind: "ok" }', () => {
    const state = deriveState({
      status: 'ok',
      models: [{ name: 'llama3.1' }, { name: 'mistral' }],
    });
    expect(state).toEqual({ kind: 'ok', models: ['llama3.1', 'mistral'] });
  });

  it('maps status=ok + empty models to { kind: "empty" }', () => {
    const state = deriveState({ status: 'ok', models: [] });
    expect(state).toEqual({ kind: 'empty' });
  });

  it('maps status=ollama_not_running to { kind: "not_running" }', () => {
    const state = deriveState({ status: 'ollama_not_running', models: [] });
    expect(state).toEqual({ kind: 'not_running' });
  });

  it('maps error flag to { kind: "error" }', () => {
    const state = deriveState(null, true);
    expect(state).toEqual({ kind: 'error' });
  });
});

describe('OllamaModelDropdown — renderState', () => {
  const noop = () => {};

  it('renders loading state with placeholder and disabled select', () => {
    const html = renderToStaticMarkup(
      renderState({ kind: 'loading' }, '', noop, false),
    );
    expect(html).toContain('Loading models');
    expect(html).toMatch(/<select[^>]*\sdisabled/);
  });

  it('renders an enabled <select> with all model options when ok', () => {
    const html = renderToStaticMarkup(
      renderState(
        { kind: 'ok', models: ['llama3.1', 'mistral'] },
        'llama3.1',
        noop,
        false,
      ),
    );
    expect(html).toContain('llama3.1');
    expect(html).toContain('mistral');
    expect(html).toMatch(/<select(?![^>]*\sdisabled)/);
  });

  it('renders "No models pulled" hint when empty', () => {
    const html = renderToStaticMarkup(
      renderState({ kind: 'empty' }, '', noop, false),
    );
    expect(html).toContain('No models pulled');
    expect(html).toContain('ollama pull llama3.1');
    expect(html).toMatch(/<select[^>]*\sdisabled/);
  });

  it('renders "Ollama not detected" hint when not_running', () => {
    const html = renderToStaticMarkup(
      renderState({ kind: 'not_running' }, '', noop, false),
    );
    expect(html).toContain('Ollama not detected');
    expect(html).toContain('brew install ollama');
  });

  it('renders "Ollama not detected" hint when fetch errors', () => {
    const html = renderToStaticMarkup(
      renderState({ kind: 'error' }, '', noop, false),
    );
    expect(html).toContain('Ollama not detected');
  });

  it('invokes onChange with the model name on select change', () => {
    const handler = vi.fn();
    // Call the onChange prop directly by traversing the returned element tree.
    const el = renderState(
      { kind: 'ok', models: ['llama3.1', 'mistral'] },
      'llama3.1',
      handler,
      false,
    );
    // The root div has the <select> as first child; invoke its onChange.
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const root = el as any;
    const selectEl = root.props.children;
    selectEl.props.onChange({ target: { value: 'mistral' } });
    expect(handler).toHaveBeenCalledWith('mistral');
  });
});
