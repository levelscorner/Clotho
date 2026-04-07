import { describe, it, expect, beforeEach } from 'vitest';
import { useHistoryStore } from '../historyStore';
import type { Node, Edge as RFEdge } from '@xyflow/react';

function makeSnapshot(id: number): { nodes: Node[]; edges: RFEdge[] } {
  return {
    nodes: [{ id: `node-${id}`, position: { x: id, y: id }, data: {} } as Node],
    edges: [],
  };
}

function resetStore() {
  useHistoryStore.getState().clear();
}

describe('historyStore', () => {
  beforeEach(() => {
    resetStore();
  });

  it('push adds snapshot to past', () => {
    const { push } = useHistoryStore.getState();
    push(makeSnapshot(1));

    const state = useHistoryStore.getState();
    expect(state.past).toHaveLength(1);
    expect(state.canUndo).toBe(true);
    expect(state.canRedo).toBe(false);
  });

  it('undo moves last past item to future', () => {
    const store = useHistoryStore.getState();
    store.push(makeSnapshot(1));
    store.push(makeSnapshot(2));

    useHistoryStore.getState().undo();

    const state = useHistoryStore.getState();
    expect(state.past).toHaveLength(1);
    expect(state.future).toHaveLength(1);
    expect(state.canUndo).toBe(true);
    expect(state.canRedo).toBe(true);
  });

  it('redo moves last future item to past', () => {
    const store = useHistoryStore.getState();
    store.push(makeSnapshot(1));
    store.push(makeSnapshot(2));

    useHistoryStore.getState().undo();
    useHistoryStore.getState().redo();

    const state = useHistoryStore.getState();
    expect(state.past).toHaveLength(2);
    expect(state.future).toHaveLength(0);
    expect(state.canUndo).toBe(true);
    expect(state.canRedo).toBe(false);
  });

  it('clear resets all stacks', () => {
    const store = useHistoryStore.getState();
    store.push(makeSnapshot(1));
    store.push(makeSnapshot(2));
    store.push(makeSnapshot(3));

    useHistoryStore.getState().clear();

    const state = useHistoryStore.getState();
    expect(state.past).toHaveLength(0);
    expect(state.future).toHaveLength(0);
    expect(state.canUndo).toBe(false);
    expect(state.canRedo).toBe(false);
  });

  it('MAX_HISTORY enforced', () => {
    const store = useHistoryStore.getState();
    for (let i = 0; i < 60; i++) {
      store.push(makeSnapshot(i));
    }

    const state = useHistoryStore.getState();
    expect(state.past.length).toBeLessThanOrEqual(50);
  });

  it('push clears future', () => {
    const store = useHistoryStore.getState();
    store.push(makeSnapshot(1));
    store.push(makeSnapshot(2));

    useHistoryStore.getState().undo();
    expect(useHistoryStore.getState().future).toHaveLength(1);

    useHistoryStore.getState().push(makeSnapshot(3));

    const state = useHistoryStore.getState();
    expect(state.future).toHaveLength(0);
    expect(state.canRedo).toBe(false);
  });
});
