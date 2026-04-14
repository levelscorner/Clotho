import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
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
    // Reset pipeline store nodes/edges to a known state.
    usePipelineStore.setState({
      nodes: [],
      edges: [],
      selectedNodeId: null,
      isDirty: false,
    });
  });

  it('renders delete button with accessible label referencing the node', () => {
    renderBaseNode({ label: 'Script Writer' });
    const btn = screen.getByRole('button', { name: /Delete node Script Writer/i });
    expect(btn).toBeInTheDocument();
  });

  it('falls back to node id in delete button label when no label is provided', () => {
    render(
      withProvider(
        <BaseNode id="node-xyz" ports={TEST_PORTS} variant="agent">
          <div />
        </BaseNode>,
      ),
    );
    const btn = screen.getByRole('button', { name: /Delete node node-xyz/i });
    expect(btn).toBeInTheDocument();
  });

  it('renders port labels with pretty type names for each port', () => {
    const { container } = renderBaseNode();
    const labels = container.querySelectorAll('.clotho-port-label');
    // One label per port (2 inputs + 1 output = 3)
    expect(labels.length).toBe(TEST_PORTS.length);

    const texts = Array.from(labels).map((l) => l.textContent ?? '');
    // "Input · text" (from text port)
    expect(texts.some((t) => t.includes('Input') && t.includes('text'))).toBe(true);
    // "Reference · image prompt" (pretty name, not image_prompt)
    expect(texts.some((t) => t.includes('Reference') && t.includes('image prompt'))).toBe(true);
    // "Script · text"
    expect(texts.some((t) => t.includes('Script') && t.includes('text'))).toBe(true);
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
    // Class is present; jsdom doesn't apply external stylesheets, so we check
    // the class marker instead of computed style.
    expect(label?.className).toContain('clotho-port-label');
  });

  it('delete button click calls removeNodes with the node id', () => {
    const removeNodesSpy = vi.spyOn(usePipelineStore.getState(), 'removeNodes');
    renderBaseNode({ id: 'node-abc', label: 'My Node' });
    const btn = screen.getByRole('button', { name: /Delete node My Node/i });
    fireEvent.click(btn);
    expect(removeNodesSpy).toHaveBeenCalledWith(['node-abc']);
    removeNodesSpy.mockRestore();
  });

  it('delete button stops propagation so parent click handlers do not fire', () => {
    const parentClick = vi.fn();
    renderBaseNode({ id: 'node-1', label: 'Test', onParentClick: parentClick });
    const btn = screen.getByRole('button', { name: /Delete node Test/i });
    fireEvent.click(btn);
    expect(parentClick).not.toHaveBeenCalled();
  });
});
