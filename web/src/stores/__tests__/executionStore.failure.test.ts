import { describe, it, expect, beforeEach } from 'vitest';
import { useExecutionStore } from '../executionStore';

// These tests exercise the store's reducer-style methods directly. The
// SSE wiring (EventSource) is browser API and untestable in jsdom, but
// updateStep is the function the SSE handlers call. Locking it behind
// tests means future SSE shape changes can't silently corrupt state.

describe('executionStore — structured failure threading', () => {
  beforeEach(() => {
    useExecutionStore.getState().reset();
  });

  it('starts with no failures and an empty stepResults map', () => {
    const s = useExecutionStore.getState();
    expect(s.stepResults.size).toBe(0);
    expect(s.status).toBeNull();
  });

  it('updateStep with status=failed + failure populates StepResult.failure', () => {
    useExecutionStore.getState().updateStep({
      node_id: 'n1',
      status: 'failed',
      error: 'Authentication failed',
      failure: {
        class: 'auth',
        stage: 'provider_call',
        retryable: false,
        message: 'Authentication failed',
        attempts: 1,
        at: '2026-04-17T12:00:00Z',
      },
    });

    const sr = useExecutionStore.getState().stepResults.get('n1');
    expect(sr).toBeDefined();
    expect(sr?.status).toBe('failed');
    expect(sr?.failure?.class).toBe('auth');
    expect(sr?.failure?.retryable).toBe(false);
    expect(sr?.error).toBe('Authentication failed');
  });

  it('updateStep merges with prior step state without dropping fields', () => {
    const store = useExecutionStore.getState();
    // First the node starts running.
    store.updateStep({ node_id: 'n1', status: 'running' });
    // Then it fails — prior fields stay; new ones merge in.
    store.updateStep({
      node_id: 'n1',
      status: 'failed',
      failure: {
        class: 'timeout',
        stage: 'provider_call',
        retryable: true,
        message: 'timed out',
        attempts: 3,
        at: '2026-04-17T12:00:01Z',
      },
    });

    const sr = useExecutionStore.getState().stepResults.get('n1');
    expect(sr?.status).toBe('failed');
    expect(sr?.failure?.class).toBe('timeout');
    expect(sr?.failure?.attempts).toBe(3);
  });

  it('updates for different nodes do not bleed into each other', () => {
    const store = useExecutionStore.getState();
    store.updateStep({
      node_id: 'a',
      status: 'failed',
      failure: {
        class: 'auth',
        stage: 'provider_call',
        retryable: false,
        message: 'auth',
        attempts: 1,
        at: '2026-04-17T12:00:00Z',
      },
    });
    store.updateStep({
      node_id: 'b',
      status: 'completed',
      output: 'ok',
    });

    const a = useExecutionStore.getState().stepResults.get('a');
    const b = useExecutionStore.getState().stepResults.get('b');
    expect(a?.failure?.class).toBe('auth');
    expect(b?.failure).toBeUndefined();
    expect(b?.output).toBe('ok');
  });

  it('reset clears stepResults including any failures', () => {
    useExecutionStore.getState().updateStep({
      node_id: 'n1',
      status: 'failed',
      failure: {
        class: 'auth',
        stage: 'provider_call',
        retryable: false,
        message: 'x',
        attempts: 1,
        at: '2026-04-17T12:00:00Z',
      },
    });
    // Re-read state — Zustand's getState() returns a snapshot, not a
    // live binding, so a previous `store` reference would be stale.
    expect(useExecutionStore.getState().stepResults.size).toBe(1);

    useExecutionStore.getState().reset();

    const after = useExecutionStore.getState();
    expect(after.stepResults.size).toBe(0);
    expect(after.status).toBeNull();
    expect(after.executionId).toBeNull();
  });
});
