import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ReliabilitySection } from '../ReliabilitySection';

// Track mock calls to setNodePin / setNodeOnFailure across tests.
const setNodePinMock = vi.fn();
const setNodeOnFailureMock = vi.fn();

// Default: a node with no prior output (pinnedOutput undefined). Tests
// re-mock per case via the makeStore helper below.
let storeState: Record<string, unknown> = {};

vi.mock('../../../../stores/pipelineStore', () => ({
  usePipelineStore: (selector: (s: unknown) => unknown) => selector(storeState),
}));

function makeStore(opts: {
  pinned?: boolean;
  pinnedOutput?: unknown;
  onFailure?: 'abort' | 'skip' | 'continue';
}) {
  storeState = {
    nodes: [
      {
        id: 'n1',
        position: { x: 0, y: 0 },
        data: {
          nodeType: 'agent',
          label: 'agent',
          ports: [],
          config: {},
          pinned: opts.pinned ?? false,
          pinnedOutput: opts.pinnedOutput,
          onFailure: opts.onFailure,
        },
      },
    ],
    setNodePin: setNodePinMock,
    setNodeOnFailure: setNodeOnFailureMock,
  };
}

describe('ReliabilitySection', () => {
  beforeEach(() => {
    setNodePinMock.mockReset();
    setNodeOnFailureMock.mockReset();
  });

  it('disables the pin checkbox when no cached output exists', () => {
    makeStore({ pinned: false, pinnedOutput: undefined });
    render(<ReliabilitySection nodeId="n1" />);
    const checkbox = screen.getByRole('checkbox') as HTMLInputElement;
    expect(checkbox.disabled).toBe(true);
    expect(checkbox.checked).toBe(false);
  });

  it('enables the pin checkbox when cached output exists', () => {
    makeStore({ pinned: false, pinnedOutput: 'cached' });
    render(<ReliabilitySection nodeId="n1" />);
    const checkbox = screen.getByRole('checkbox') as HTMLInputElement;
    expect(checkbox.disabled).toBe(false);
  });

  it('clicking pin checkbox calls setNodePin', () => {
    makeStore({ pinned: false, pinnedOutput: 'cached' });
    render(<ReliabilitySection nodeId="n1" />);
    const checkbox = screen.getByRole('checkbox');
    fireEvent.click(checkbox);
    expect(setNodePinMock).toHaveBeenCalledWith('n1', true);
  });

  it('shows pinned-state helper when output is cached', () => {
    makeStore({ pinned: true, pinnedOutput: 'cached' });
    render(<ReliabilitySection nodeId="n1" />);
    expect(
      screen.getByText(/Engine skips this node and serves the cached output/),
    ).toBeInTheDocument();
  });

  it('shows the explainer when no cached output yet', () => {
    makeStore({ pinned: false, pinnedOutput: undefined });
    render(<ReliabilitySection nodeId="n1" />);
    expect(
      screen.getByText(/Run the pipeline once first/),
    ).toBeInTheDocument();
  });

  it('on-failure dropdown defaults to abort when none set', () => {
    makeStore({});
    render(<ReliabilitySection nodeId="n1" />);
    const select = screen.getByRole('combobox') as HTMLSelectElement;
    expect(select.value).toBe('abort');
  });

  it('changing on-failure dropdown calls setNodeOnFailure', () => {
    makeStore({ onFailure: 'abort' });
    render(<ReliabilitySection nodeId="n1" />);
    const select = screen.getByRole('combobox') as HTMLSelectElement;
    fireEvent.change(select, { target: { value: 'skip' } });
    expect(setNodeOnFailureMock).toHaveBeenCalledWith('n1', 'skip');
  });

  it('renders nothing when the node id is unknown', () => {
    storeState = {
      nodes: [],
      setNodePin: setNodePinMock,
      setNodeOnFailure: setNodeOnFailureMock,
    };
    const { container } = render(<ReliabilitySection nodeId="missing" />);
    expect(container.firstChild).toBeNull();
  });
});
