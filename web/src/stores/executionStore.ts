import { create } from 'zustand';
import type { Edge as RFEdge } from '@xyflow/react';
import type { ExecutionStatus, StepResult } from '../lib/types';
import { api } from '../lib/api';

// ---------------------------------------------------------------------------
// Selector helper: downstream-of-failed node ids
//
// Given the current stepResults map and the pipeline edges, BFS forward from
// every failed node and return the set of reachable node ids (NOT including
// the failed nodes themselves). These are the nodes the UI should render as
// "blocked" — they can never succeed while their upstream dependency is red.
// ---------------------------------------------------------------------------

export function computeBlockedNodeIds(
  stepResults: Map<string, StepResult>,
  edges: ReadonlyArray<RFEdge>,
): Set<string> {
  const failed: string[] = [];
  stepResults.forEach((result, nodeId) => {
    if (result.status === 'failed') {
      failed.push(nodeId);
    }
  });

  if (failed.length === 0) {
    return new Set();
  }

  // Build adjacency map source → [targets]
  const adj = new Map<string, string[]>();
  for (const edge of edges) {
    const list = adj.get(edge.source);
    if (list) {
      list.push(edge.target);
    } else {
      adj.set(edge.source, [edge.target]);
    }
  }

  const blocked = new Set<string>();
  const queue: string[] = [...failed];
  const visited = new Set<string>(failed);

  while (queue.length > 0) {
    const current = queue.shift() as string;
    const targets = adj.get(current);
    if (!targets) continue;
    for (const target of targets) {
      if (visited.has(target)) continue;
      visited.add(target);
      blocked.add(target);
      queue.push(target);
    }
  }

  return blocked;
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

interface ExecutionState {
  executionId: string | null;
  status: ExecutionStatus | null;
  stepResults: Map<string, StepResult>;
  totalCost: number;
  isStreaming: boolean;

  startExecution: (pipelineId: string, fromNodeId?: string) => Promise<void>;
  retryNode: (executionId: string, nodeId: string) => void;
  connectSSE: (executionId: string) => void;
  disconnectSSE: () => void;
  updateStep: (result: StepResult) => void;
  appendChunk: (nodeId: string, chunk: string) => void;
  reset: () => void;
}

let eventSource: EventSource | null = null;

export const useExecutionStore = create<ExecutionState>((set, get) => ({
  executionId: null,
  status: null,
  stepResults: new Map(),
  totalCost: 0,
  isStreaming: false,

  startExecution: async (pipelineId, fromNodeId) => {
    get().disconnectSSE();

    set({
      executionId: null,
      status: 'pending',
      stepResults: new Map(),
      totalCost: 0,
      isStreaming: false,
    });

    const body = fromNodeId ? { from_node_id: fromNodeId } : undefined;
    const execution = await api.post<{ id: string }>(
      `/pipelines/${pipelineId}/execute`,
      body,
    );

    set({ executionId: execution.id, status: 'running' });
    get().connectSSE(execution.id);
  },

  retryNode: (executionId, nodeId) => {
    // Clear the failed step result so UI resets
    set((state) => {
      const next = new Map(state.stepResults);
      next.delete(nodeId);
      return { stepResults: next, status: 'running' };
    });
    // Re-run from the failed node via the existing execution's pipeline
    // The actual re-execution is handled by startExecution with from_node_id
    const pipelineId = executionId; // Will be replaced by proper pipeline lookup
    get().startExecution(pipelineId, nodeId).catch(() => {
      set({ status: 'failed' });
    });
  },

  connectSSE: (executionId) => {
    if (eventSource) {
      eventSource.close();
    }

    const es = new EventSource(`/api/executions/${executionId}/stream`);
    eventSource = es;
    set({ isStreaming: true });

    es.addEventListener('step_started', (e: MessageEvent) => {
      const data = JSON.parse(e.data) as { node_id: string };
      get().updateStep({
        node_id: data.node_id,
        status: 'running',
      });
    });

    es.addEventListener('step_chunk', (e: MessageEvent) => {
      const data = JSON.parse(e.data) as {
        node_id: string;
        chunk: string;
      };
      get().appendChunk(data.node_id, data.chunk);
    });

    es.addEventListener('step_completed', (e: MessageEvent) => {
      const data = JSON.parse(e.data) as StepResult;
      get().updateStep({ ...data, status: 'completed' });
    });

    es.addEventListener('step_failed', (e: MessageEvent) => {
      const data = JSON.parse(e.data) as StepResult;
      get().updateStep({ ...data, status: 'failed' });
    });

    es.addEventListener('execution_completed', (e: MessageEvent) => {
      const data = JSON.parse(e.data) as { total_cost?: number };
      set({
        status: 'completed',
        totalCost: data.total_cost ?? get().totalCost,
        isStreaming: false,
      });
      get().disconnectSSE();
    });

    es.addEventListener('execution_failed', (e: MessageEvent) => {
      const data = JSON.parse(e.data) as { error?: string };
      void data;
      set({ status: 'failed', isStreaming: false });
      get().disconnectSSE();
    });

    es.onerror = () => {
      set({ isStreaming: false });
      get().disconnectSSE();
    };
  },

  disconnectSSE: () => {
    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }
    set({ isStreaming: false });
  },

  updateStep: (result) => {
    set((state) => {
      const next = new Map(state.stepResults);
      const prev = next.get(result.node_id);
      next.set(result.node_id, { ...prev, ...result } as StepResult);

      let cost = 0;
      next.forEach((sr) => {
        cost += sr.cost ?? 0;
      });

      return { stepResults: next, totalCost: cost };
    });
  },

  appendChunk: (nodeId, chunk) => {
    set((state) => {
      const next = new Map(state.stepResults);
      const prev = next.get(nodeId);
      const currentOutput = prev?.output ?? '';
      next.set(nodeId, {
        ...(prev ?? { node_id: nodeId, status: 'running' as const }),
        output: currentOutput + chunk,
      });
      return { stepResults: next };
    });
  },

  reset: () => {
    get().disconnectSSE();
    set({
      executionId: null,
      status: null,
      stepResults: new Map(),
      totalCost: 0,
      isStreaming: false,
    });
  },
}));
