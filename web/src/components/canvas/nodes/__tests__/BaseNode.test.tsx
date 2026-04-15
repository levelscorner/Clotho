import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ReactFlowProvider } from '@xyflow/react';
import type { ReactElement } from 'react';
import { BaseNode } from '../BaseNode';
import { usePipelineStore } from '../../../../stores/pipelineStore';
import { useExecutionStore } from '../../../../stores/executionStore';
import type { Port } from '../../../../lib/types';

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const TEST_PORTS: Port[] = [
  { id: 'in-1', name: 'Input', type: 'text', direction: 'input', required: true },
  { id: 'in-2', name: 'Reference', type: 'image_prompt', direction: 'input', required: false },
  { id: 'out-1', name: 'Script', type: 'text', direction: 'output', required: false },
];

function withProvider(ui: ReactElement) {
  return <ReactFlowProvider>{ui}</ReactFlowProvider>;
}

function renderBaseNode(
  overrides: {
    id?: string;
    label?: string;
    onParentClick?: () => void;
  } = {},
) {
  const { id = 'node-test', label = 'Test Agent', onParentClick } = overrides;
  return render(
    withProvider(
      <div onClick={onParentClick}>
        <BaseNode id={id} ports={TEST_PORTS} variant="agent" label={label}>
          <div className="clotho-node__header">{label}</div>
        </BaseNode>
      </div>,
    ),
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('BaseNode', () => {
  beforeEach(() => {
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
      selectedNodeId: null,
      isDirty: false,
      lockedNodes: new Set<string>(),
      renamingNodeId: null,
    });
  });

  it('renders actions menu trigger with an accessible label referencing the node', () => {
    renderBaseNode({ label: 'Script Writer' });
    const btn = screen.getByRole('button', { name: /Actions for Script Writer/i });
    expect(btn).toBeInTheDocument();
  });

  it('falls back to generic "Node actions" trigger label when no label is provided', () => {
    render(
      withProvider(
        <BaseNode id="node-xyz" ports={TEST_PORTS} variant="agent">
          <div />
        </BaseNode>,
      ),
    );
    const btn = screen.getByRole('button', { name: /Node actions/i });
    expect(btn).toBeInTheDocument();
  });

  it('applies clotho-handle--{type} class to each port handle', () => {
    const { container } = renderBaseNode();
    expect(container.querySelector('.clotho-handle--text')).toBeTruthy();
    expect(container.querySelector('.clotho-handle--image_prompt')).toBeTruthy();
    const handles = container.querySelectorAll('.react-flow__handle');
    handles.forEach((h) => {
      expect(h.className).toContain('clotho-handle');
    });
  });

  it('renders port labels with name plus required-asterisk only (type hidden in text)', () => {
    const { container } = renderBaseNode();
    const labels = container.querySelectorAll('.clotho-port-label');
    expect(labels.length).toBe(TEST_PORTS.length);

    const texts = Array.from(labels).map((l) => l.textContent ?? '');
    expect(texts).toContain('Input*');
    expect(texts).toContain('Reference');
    expect(texts).toContain('Script');

    texts.forEach((t) => {
      expect(t).not.toMatch(/image[_ ]prompt/);
      expect(t).not.toContain('·');
    });
  });

  it('exposes the port type via tooltip on the label for power users', () => {
    const { container } = renderBaseNode();
    const labels = container.querySelectorAll('.clotho-port-label');
    const titles = Array.from(labels).map((l) => l.getAttribute('title'));
    expect(titles).toContain('text');
    expect(titles).toContain('image prompt');
  });

  it('splits port labels across input and output sides', () => {
    const { container } = renderBaseNode();
    const inLabels = container.querySelectorAll('.clotho-port-label--in');
    const outLabels = container.querySelectorAll('.clotho-port-label--out');
    expect(inLabels.length).toBe(2);
    expect(outLabels.length).toBe(1);
  });

  it('port labels are hidden by default (opacity 0 via base class)', () => {
    const { container } = renderBaseNode();
    const label = container.querySelector('.clotho-port-label');
    expect(label).toBeTruthy();
    expect(label?.className).toContain('clotho-port-label');
  });

  it('renders a lock badge only when the node is in lockedNodes', () => {
    const { container, rerender } = renderBaseNode({ id: 'node-lock', label: 'Locked' });
    // Initially unlocked — no badge
    expect(container.querySelector('.clotho-node__lock-badge')).toBeNull();

    usePipelineStore.setState({ lockedNodes: new Set(['node-lock']) });
    rerender(
      withProvider(
        <BaseNode id="node-lock" ports={TEST_PORTS} variant="agent" label="Locked">
          <div />
        </BaseNode>,
      ),
    );
    expect(container.querySelector('.clotho-node__lock-badge')).toBeTruthy();
  });
});
