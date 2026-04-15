import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { NodeActionsMenu } from '../NodeActionsMenu';
import { usePipelineStore } from '../../../../stores/pipelineStore';
import { useExecutionStore } from '../../../../stores/executionStore';
import type { StepResult } from '../../../../lib/types';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// Radix DropdownMenu opens on pointerdown (not plain click) and animates on
// a microtask. We drive it via the fireEvent pointer sequence and flush any
// pending timers/microtasks inside `act` so the portal content lands.
async function openMenu(nodeId = 'node-x', label = 'Agent'): Promise<void> {
  render(<NodeActionsMenu nodeId={nodeId} label={label} />);
  const trigger = screen.getByRole('button', { name: /Actions for Agent/i });
  await act(async () => {
    fireEvent.pointerDown(trigger, { button: 0, pointerType: 'mouse' });
    fireEvent.pointerUp(trigger, { button: 0, pointerType: 'mouse' });
    fireEvent.click(trigger);
  });
}

function setStep(nodeId: string, output: string | undefined): void {
  const step: StepResult = {
    node_id: nodeId,
    status: 'completed',
    output,
    started_at: new Date().toISOString(),
    completed_at: new Date().toISOString(),
    tokens_used: 0,
    cost: 0,
  };
  useExecutionStore.setState({
    stepResults: new Map([[nodeId, step]]),
  });
}

// ---------------------------------------------------------------------------
// Fresh store state between tests
// ---------------------------------------------------------------------------

beforeEach(() => {
  usePipelineStore.setState({
    nodes: [],
    edges: [],
    selectedNodeId: null,
    isDirty: false,
    lockedNodes: new Set<string>(),
    renamingNodeId: null,
  });
  useExecutionStore.setState({
    executionId: null,
    status: null,
    stepResults: new Map(),
    totalCost: 0,
    isStreaming: false,
  });
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('NodeActionsMenu', () => {
  it('renders the ⋯ trigger with accessible label', () => {
    render(<NodeActionsMenu nodeId="n1" label="Script Writer" />);
    const trigger = screen.getByRole('button', { name: /Actions for Script Writer/i });
    expect(trigger).toBeInTheDocument();
  });

  it('opens the menu with the core items (Duplicate, Rename, Lock, Delete)', async () => {
    await openMenu('n1', 'Agent');
    expect(screen.getByText('Duplicate')).toBeInTheDocument();
    expect(screen.getByText('Rename')).toBeInTheDocument();
    expect(screen.getByText('Lock')).toBeInTheDocument();
    expect(screen.getByText('Delete')).toBeInTheDocument();
  });

  it('does not render Download/Reveal items when the node has no output', async () => {
    await openMenu('n1', 'Agent');
    expect(screen.queryByText(/Download output/i)).toBeNull();
    expect(screen.queryByText(/Reveal in folder/i)).toBeNull();
  });

  it('renders both Download and Reveal when output is a clotho://file reference', async () => {
    setStep('n1', 'clotho://file/generated/out.png');
    await openMenu('n1', 'Agent');
    expect(screen.getByText(/Download output/i)).toBeInTheDocument();
    expect(screen.getByText(/Reveal in folder/i)).toBeInTheDocument();
  });

  it('renders only Download (not Reveal) when output is a plain URL/data URI', async () => {
    setStep('n1', 'https://example.com/out.png');
    await openMenu('n1', 'Agent');
    expect(screen.getByText(/Download output/i)).toBeInTheDocument();
    expect(screen.queryByText(/Reveal in folder/i)).toBeNull();
  });

  it('shows "Unlock" in place of "Lock" and marks destructive items disabled when locked', async () => {
    usePipelineStore.setState({ lockedNodes: new Set(['n1']) });
    await openMenu('n1', 'Agent');
    expect(screen.getByText('Unlock')).toBeInTheDocument();
    expect(screen.queryByText('Lock')).toBeNull();

    const duplicate = screen.getByText('Duplicate').closest('[role="menuitem"]');
    const del = screen.getByText('Delete').closest('[role="menuitem"]');
    expect(duplicate?.getAttribute('data-disabled')).not.toBeNull();
    expect(del?.getAttribute('data-disabled')).not.toBeNull();
  });

  it('invokes toggleLock when the Lock item is selected', async () => {
    const toggleLockSpy = vi.spyOn(usePipelineStore.getState(), 'toggleLock');
    await openMenu('n1', 'Agent');
    fireEvent.click(screen.getByText('Lock'));
    expect(toggleLockSpy).toHaveBeenCalledWith('n1');
    toggleLockSpy.mockRestore();
  });

  it('invokes removeNodes([nodeId]) when Delete is selected on an unlocked node', async () => {
    const removeNodesSpy = vi.spyOn(usePipelineStore.getState(), 'removeNodes');
    await openMenu('n1', 'Agent');
    fireEvent.click(screen.getByText('Delete'));
    expect(removeNodesSpy).toHaveBeenCalledWith(['n1']);
    removeNodesSpy.mockRestore();
  });

  it('invokes duplicateNode(nodeId) when Duplicate is selected on an unlocked node', async () => {
    const dupSpy = vi.spyOn(usePipelineStore.getState(), 'duplicateNode');
    await openMenu('n1', 'Agent');
    fireEvent.click(screen.getByText('Duplicate'));
    expect(dupSpy).toHaveBeenCalledWith('n1');
    dupSpy.mockRestore();
  });
});
