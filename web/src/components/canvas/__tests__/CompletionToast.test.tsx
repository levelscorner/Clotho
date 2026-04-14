import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { CompletionToast } from '../CompletionToast';
import {
  useExecutionStore,
  computeBlockedNodeIds,
} from '../../../stores/executionStore';
import { usePipelineStore } from '../../../stores/pipelineStore';
import type { StepResult } from '../../../lib/types';

function resetStores(): void {
  useExecutionStore.setState({
    executionId: null,
    status: null,
    stepResults: new Map(),
    totalCost: 0,
    isStreaming: false,
  });
  usePipelineStore.setState({
    nodes: [],
    edges: [],
    viewport: { x: 0, y: 0, zoom: 1 },
    pipelineId: null,
    pipelineName: 'Untitled Pipeline',
    isDirty: false,
    selectedNodeId: null,
  });
}

describe('CompletionToast', () => {
  beforeEach(() => {
    resetStores();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders nothing before a pipeline completes', () => {
    const html = renderToStaticMarkup(<CompletionToast />);
    expect(html).toBe('');
  });

  it('renders nothing when status is completed but there is a failure', () => {
    const results = new Map<string, StepResult>();
    results.set('a', { node_id: 'a', status: 'failed', error: 'boom' });
    useExecutionStore.setState({ status: 'completed', stepResults: results });
    const html = renderToStaticMarkup(<CompletionToast />);
    expect(html).toBe('');
  });
});

describe('computeBlockedNodeIds', () => {
  it('returns empty set when no nodes have failed', () => {
    const results = new Map<string, StepResult>();
    results.set('a', { node_id: 'a', status: 'completed' });
    const blocked = computeBlockedNodeIds(results, []);
    expect(blocked.size).toBe(0);
  });

  it('returns all downstream nodes of a failed node', () => {
    const results = new Map<string, StepResult>();
    results.set('b', { node_id: 'b', status: 'failed' });
    const edges = [
      { id: 'e1', source: 'a', target: 'b' } as const,
      { id: 'e2', source: 'b', target: 'c' } as const,
      { id: 'e3', source: 'c', target: 'd' } as const,
    ];
    const blocked = computeBlockedNodeIds(
      results,
      edges.map((e) => ({
        ...e,
        source: e.source,
        target: e.target,
      })) as unknown as never,
    );
    expect(blocked.has('c')).toBe(true);
    expect(blocked.has('d')).toBe(true);
    expect(blocked.has('a')).toBe(false);
    expect(blocked.has('b')).toBe(false);
  });

  it('handles multiple failed nodes with overlapping downstreams', () => {
    const results = new Map<string, StepResult>();
    results.set('a', { node_id: 'a', status: 'failed' });
    results.set('b', { node_id: 'b', status: 'failed' });
    const edges = [
      { id: 'e1', source: 'a', target: 'c' },
      { id: 'e2', source: 'b', target: 'c' },
      { id: 'e3', source: 'c', target: 'd' },
    ];
    const blocked = computeBlockedNodeIds(
      results,
      edges as unknown as never,
    );
    expect(blocked.has('c')).toBe(true);
    expect(blocked.has('d')).toBe(true);
  });
});
