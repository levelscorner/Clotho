import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { TestStepButton } from '../TestStepButton';

const testNodeMock = vi.fn();

let storeState: Record<string, unknown> = {};
vi.mock('../../../stores/pipelineStore', () => ({
  usePipelineStore: (selector: (s: unknown) => unknown) => selector(storeState),
}));

vi.mock('../../../lib/api', () => ({
  testNode: (...args: unknown[]) => testNodeMock(...args),
  api: { post: vi.fn(), get: vi.fn() },
}));

function makeNode() {
  storeState = {
    nodes: [
      {
        id: 'n1',
        position: { x: 0, y: 0 },
        data: {
          nodeType: 'agent',
          label: 'agent',
          ports: [],
          config: { provider: 'ollama', model: 'llama3.1' },
        },
      },
    ],
  };
}

describe('TestStepButton', () => {
  beforeEach(() => {
    testNodeMock.mockReset();
  });

  it('renders nothing when the node is missing', () => {
    storeState = { nodes: [] };
    const { container } = render(<TestStepButton nodeId="missing" />);
    expect(container.firstChild).toBeNull();
  });

  it('shows the Test button by default', () => {
    makeNode();
    render(<TestStepButton nodeId="n1" />);
    expect(screen.getByText('Test step in isolation')).toBeInTheDocument();
  });

  it('clicking Test calls testNode with the live node payload', async () => {
    makeNode();
    testNodeMock.mockResolvedValue({ duration_ms: 50, output: 'hi' });

    render(<TestStepButton nodeId="n1" />);
    fireEvent.click(screen.getByText('Test step in isolation'));

    await waitFor(() => expect(testNodeMock).toHaveBeenCalledTimes(1));
    const arg = testNodeMock.mock.calls[0][0];
    expect(arg.node.id).toBe('n1');
    expect(arg.node.type).toBe('agent');
    expect(arg.inputs).toEqual({});
  });

  it('renders OK + duration + output preview on success', async () => {
    makeNode();
    testNodeMock.mockResolvedValue({
      duration_ms: 123,
      output: 'lighthouse haiku',
      tokens_used: 42,
    });

    render(<TestStepButton nodeId="n1" />);
    fireEvent.click(screen.getByText('Test step in isolation'));

    await waitFor(() => expect(screen.getByText(/OK · 123ms/)).toBeInTheDocument());
    expect(screen.getByText(/42 tokens/)).toBeInTheDocument();
    expect(screen.getByText(/lighthouse haiku/)).toBeInTheDocument();
  });

  it('renders Failed badge + Why? button when failure returned', async () => {
    makeNode();
    testNodeMock.mockResolvedValue({
      duration_ms: 10,
      failure: {
        class: 'auth',
        stage: 'provider_call',
        retryable: false,
        message: 'auth failed',
        attempts: 1,
        at: '2026-04-17T12:00:00Z',
      },
    });

    render(<TestStepButton nodeId="n1" />);
    fireEvent.click(screen.getByText('Test step in isolation'));

    await waitFor(() => expect(screen.getByText(/Failed · 10ms/)).toBeInTheDocument());
    expect(screen.getByText(/Why did it fail\?/)).toBeInTheDocument();
  });

  it('shows raw error text when no structured failure available', async () => {
    makeNode();
    testNodeMock.mockResolvedValue({
      duration_ms: 5,
      error: 'network kaboom',
    });

    render(<TestStepButton nodeId="n1" />);
    fireEvent.click(screen.getByText('Test step in isolation'));

    // Wait for the result row to render — the duration string is the
    // most stable anchor since "network kaboom" might appear in
    // multiple places.
    await waitFor(() => expect(screen.getByText(/Failed · 5ms/)).toBeInTheDocument());
    expect(screen.getByText('network kaboom')).toBeInTheDocument();
  });

  it('handles a thrown rejection by surfacing an inline error', async () => {
    makeNode();
    testNodeMock.mockRejectedValue(new Error('500 Internal Server Error'));

    render(<TestStepButton nodeId="n1" />);
    fireEvent.click(screen.getByText('Test step in isolation'));

    await waitFor(() => expect(screen.getByText(/Failed/)).toBeInTheDocument());
    expect(screen.getByText('500 Internal Server Error')).toBeInTheDocument();
  });
});
