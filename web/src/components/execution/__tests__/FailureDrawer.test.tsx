import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import {
  FailureDrawer,
  failureTooltipText,
  firstFailureFromMap,
} from '../FailureDrawer';
import type { StepFailure, StepResult } from '../../../lib/types';

// Stub the executionStore — drawer reads executionId + retryNode from it.
vi.mock('../../../stores/executionStore', () => ({
  useExecutionStore: (selector: (s: unknown) => unknown) =>
    selector({
      executionId: 'exec-123',
      retryNode: vi.fn(),
    }),
}));

const baseFailure: StepFailure = {
  class: 'auth',
  stage: 'provider_call',
  provider: 'openai',
  model: 'gpt-4o',
  retryable: false,
  message: 'Authentication failed',
  cause: '401 Unauthorized',
  hint: 'Verify the API key in Settings.',
  attempts: 1,
  at: '2026-04-17T12:00:00Z',
};

describe('FailureDrawer', () => {
  beforeEach(() => {
    // Wipe clipboard between tests; some assertions check it.
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
      writable: true,
      configurable: true,
    });
  });

  it('renders the class badge, message, hint, and provider/model', () => {
    render(
      <FailureDrawer nodeId="n1" failure={baseFailure} onClose={() => undefined} />,
    );

    expect(screen.getByText('Auth')).toBeInTheDocument();
    expect(screen.getByText('Authentication failed')).toBeInTheDocument();
    expect(screen.getByText(/Verify the API key/)).toBeInTheDocument();
    expect(screen.getByText(/openai/)).toBeInTheDocument();
  });

  it('hides the Rerun button when failure is non-retryable', () => {
    render(
      <FailureDrawer nodeId="n1" failure={baseFailure} onClose={() => undefined} />,
    );
    expect(screen.queryByText(/Rerun from this node/)).toBeNull();
  });

  it('shows the Rerun button when failure is retryable', () => {
    render(
      <FailureDrawer
        nodeId="n1"
        failure={{ ...baseFailure, retryable: true }}
        onClose={() => undefined}
      />,
    );
    expect(screen.getByText(/Rerun from this node/)).toBeInTheDocument();
  });

  it('Copy diagnostic writes JSON to the clipboard', async () => {
    render(
      <FailureDrawer nodeId="n1" failure={baseFailure} onClose={() => undefined} />,
    );

    const copyBtn = screen.getByText(/Copy diagnostic/);
    fireEvent.click(copyBtn);

    // Yield so the async write resolves.
    await Promise.resolve();
    await Promise.resolve();

    const writeText = navigator.clipboard.writeText as ReturnType<typeof vi.fn>;
    expect(writeText).toHaveBeenCalledTimes(1);
    const arg = writeText.mock.calls[0][0] as string;
    const parsed = JSON.parse(arg);
    expect(parsed.execution_id).toBe('exec-123');
    expect(parsed.node_id).toBe('n1');
    expect(parsed.failure.class).toBe('auth');
  });

  it('clicking the close button calls onClose', () => {
    const onClose = vi.fn();
    render(
      <FailureDrawer nodeId="n1" failure={baseFailure} onClose={onClose} />,
    );
    fireEvent.click(screen.getByLabelText('Close'));
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});

describe('failureTooltipText', () => {
  it('renders class · provider · attempts', () => {
    const got = failureTooltipText(baseFailure);
    expect(got).toContain('Auth');
    expect(got).toContain('openai');
    expect(got).toContain('attempt 1');
  });

  it('drops attempts segment when zero', () => {
    const got = failureTooltipText({ ...baseFailure, attempts: 0 });
    expect(got).not.toContain('attempt');
  });

  it('drops provider segment when missing', () => {
    const got = failureTooltipText({ ...baseFailure, provider: undefined });
    expect(got).not.toContain('openai');
  });
});

describe('firstFailureFromMap', () => {
  it('returns null when no step has a failure', () => {
    const map = new Map<string, StepResult>([
      ['a', { node_id: 'a', status: 'completed' }],
    ]);
    expect(firstFailureFromMap(map)).toBeNull();
  });

  it('returns the first failed step with a structured failure', () => {
    const map = new Map<string, StepResult>([
      ['a', { node_id: 'a', status: 'completed' }],
      ['b', { node_id: 'b', status: 'failed', failure: baseFailure }],
      ['c', { node_id: 'c', status: 'failed', failure: baseFailure }],
    ]);
    const got = firstFailureFromMap(map);
    expect(got?.nodeId).toBe('b');
    expect(got?.failure.class).toBe('auth');
  });

  it('skips failed steps without a structured failure (legacy string-only)', () => {
    const map = new Map<string, StepResult>([
      ['a', { node_id: 'a', status: 'failed', error: 'old' }], // no failure field
    ]);
    expect(firstFailureFromMap(map)).toBeNull();
  });
});
