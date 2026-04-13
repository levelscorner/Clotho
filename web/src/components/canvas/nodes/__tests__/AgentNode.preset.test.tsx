import { describe, it, expect, beforeEach } from 'vitest';
import { render } from '@testing-library/react';
import { ReactFlowProvider, type NodeProps, type Node } from '@xyflow/react';
import type { ReactElement } from 'react';
import { AgentNode } from '../AgentNode';
import { useExecutionStore } from '../../../../stores/executionStore';
import type { AgentNodeData, StepResult } from '../../../../lib/types';
import { getFixture } from '../__mocks__/node-fixtures';

// React Flow's NodeProps has many internal fields we don't exercise.
// We build the minimum shape the component reads and cast.
type AgentNodeType = Node<AgentNodeData>;

function withProvider(ui: ReactElement) {
  return <ReactFlowProvider>{ui}</ReactFlowProvider>;
}

function renderFixture(
  category: 'script' | 'crafter' | 'generic',
  state: 'queued' | 'running' | 'complete' | 'empty-complete' | 'failed',
): string {
  const fixture = getFixture(category, state);
  if (fixture.stepResult) {
    seedStepResult(fixture.id, fixture.stepResult);
  }
  const props = {
    id: fixture.id,
    data: fixture.data,
    selected: fixture.selected,
    type: 'agent',
    dragging: false,
    isConnectable: true,
    positionAbsoluteX: 0,
    positionAbsoluteY: 0,
    zIndex: 0,
    deletable: true,
    selectable: true,
    draggable: true,
  } as unknown as NodeProps<AgentNodeType>;
  const { container } = render(withProvider(<AgentNode {...props} />));
  return container.innerHTML;
}

function seedStepResult(id: string, result: StepResult): void {
  useExecutionStore.setState({
    stepResults: new Map([[id, result]]),
  });
}

function renderUnknownPreset(): string {
  const fixture = getFixture('generic', 'complete');
  const data: AgentNodeData = {
    ...fixture.data,
    config: {
      ...fixture.data.config,
      preset_category: 'totally-unknown-value',
    },
  };
  if (fixture.stepResult) {
    seedStepResult(fixture.id, fixture.stepResult);
  }
  const props = {
    id: fixture.id,
    data,
    selected: false,
    type: 'agent',
    dragging: false,
    isConnectable: true,
    positionAbsoluteX: 0,
    positionAbsoluteY: 0,
    zIndex: 0,
    deletable: true,
    selectable: true,
    draggable: true,
  } as unknown as NodeProps<AgentNodeType>;
  const { container } = render(withProvider(<AgentNode {...props} />));
  return container.innerHTML;
}

describe('AgentNode preset dispatch', () => {
  beforeEach(() => {
    useExecutionStore.setState({
      executionId: null,
      status: null,
      stepResults: new Map(),
      totalCost: 0,
      isStreaming: false,
    });
  });

  it('renders script modifier class for preset_category="script"', () => {
    const html = renderFixture('script', 'complete');
    expect(html).toContain('clotho-node--agent-script');
    expect(html).not.toContain('clotho-node--agent-crafter');
    expect(html).not.toContain('clotho-node--agent-generic');
  });

  it('renders crafter modifier class for preset_category="crafter"', () => {
    const html = renderFixture('crafter', 'complete');
    expect(html).toContain('clotho-node--agent-crafter');
    expect(html).not.toContain('clotho-node--agent-script');
  });

  it('falls back to generic when preset_category is undefined', () => {
    const html = renderFixture('generic', 'complete');
    expect(html).toContain('clotho-node--agent-generic');
    expect(html).not.toContain('clotho-node--agent-script');
    expect(html).not.toContain('clotho-node--agent-crafter');
  });

  it('falls back to generic when preset_category is an unknown string', () => {
    const html = renderUnknownPreset();
    expect(html).toContain('clotho-node--agent-generic');
  });

  it('renders token count in script readout when tokens_used is present', () => {
    const html = renderFixture('script', 'complete');
    expect(html).toContain('clotho-node__script-readout');
    expect(html).toContain('142 tokens');
  });

  it('renders em-dash placeholder in script readout when tokens are absent', () => {
    const html = renderFixture('script', 'queued');
    expect(html).toContain('clotho-node__script-readout');
    expect(html).toContain('\u2014'); // em-dash
  });

  it('preserves aria-label on the node (F-006 regression guard)', () => {
    const html = renderFixture('script', 'complete');
    expect(html).toMatch(/aria-label="[^"]*Script Writer[^"]*agent[^"]*"/);
  });
});
