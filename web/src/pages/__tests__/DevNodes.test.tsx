import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';

// CSS side-effect imports are stubbed — vitest's node environment can't
// parse raw .css files.
vi.mock('../DevNodes.css', () => ({}));

import DevNodes from '../DevNodes';
import { useExecutionStore } from '../../stores/executionStore';
import { ALL_FIXTURES } from '../../components/canvas/nodes/__mocks__/node-fixtures';

describe('DevNodes testbed page', () => {
  beforeEach(() => {
    useExecutionStore.setState({
      executionId: null,
      status: null,
      stepResults: new Map(),
      totalCost: 0,
      isStreaming: false,
    });
  });

  it('exposes fixtures for every node kind', () => {
    expect(ALL_FIXTURES.agent.length).toBeGreaterThanOrEqual(5);
    expect(ALL_FIXTURES.media.length).toBeGreaterThanOrEqual(15);
    expect(ALL_FIXTURES.tool.length).toBeGreaterThanOrEqual(5);
  });

  it('renders the page shell with header + all tabs', () => {
    const html = renderToStaticMarkup(<DevNodes />);
    expect(html).toContain('Clotho');
    expect(html).toContain('Node Fixtures');
    // The default active tab is agent — all tab labels should appear
    // in the tab bar regardless.
    expect(html).toContain('Agent');
    expect(html).toContain('Media · Image');
    expect(html).toContain('Media · Video');
    expect(html).toContain('Media · Audio');
    expect(html).toContain('Tool');
  });

  it('renders at least one fixture card on the default tab', () => {
    const html = renderToStaticMarkup(<DevNodes />);
    // The default tab is agent. Our agent fixtures use the id pattern
    // fixture-agent-<state>.
    expect(html).toMatch(/fixture-agent-/);
    // Confirm at least one state badge rendered.
    expect(html).toMatch(/QUEUED|RUNNING|COMPLETE|FAILED/);
  });

  it('includes a fixture for every state across all kinds', () => {
    // Not a full snapshot — just guards that the fixtures cover the matrix.
    const states = ['queued', 'running', 'complete', 'empty-complete', 'failed'];
    for (const state of states) {
      expect(ALL_FIXTURES.agent.some((f) => f.state === state)).toBe(true);
      expect(ALL_FIXTURES.media.some((f) => f.state === state)).toBe(true);
      expect(ALL_FIXTURES.tool.some((f) => f.state === state)).toBe(true);
    }
  });
});
