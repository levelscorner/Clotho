import { create } from 'zustand';
import type { Edge as RFEdge } from '@xyflow/react';
import type { ExecutionStatus, StepResult, StepFailure } from '../lib/types';
import { api } from '../lib/api';
import { parseEnvelope } from './sseParse';
import { coerceStepFailure } from '../lib/failureSchema';

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
  /**
   * Relative path (forward-slashed) under the backend's DataDir that
   * contains this execution's manifest.json, agent .txt files, and
   * media outputs. Populated on execution_completed so the UI can
   * offer an "Open folder" action. Null until the run finishes.
   */
  artifactDir: string | null;

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
  artifactDir: null,

  startExecution: async (pipelineId, fromNodeId) => {
    get().disconnectSSE();

    set({
      executionId: null,
      status: 'pending',
      stepResults: new Map(),
      totalCost: 0,
      isStreaming: false,
      artifactDir: null,
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

    // SSE event envelope shape now lives in ./sseParse as BaseEnvelope.
    // parseEnvelope returns a typed view or null on malformed input, so
    // the listeners below never see arbitrary parsed JSON.

    const stringify = (value: unknown): string => {
      if (value == null) return '';
      if (typeof value === 'string') return value;
      // For media nodes, output is typically a data URI string already; for
      // agent nodes, it's also a string. If a provider returns a structured
      // object, we serialise it so the Inspector can display something.
      try {
        return JSON.stringify(value);
      } catch {
        return String(value);
      }
    };

    es.addEventListener('step_started', (e: MessageEvent) => {
      const env = parseEnvelope(e.data);
      if (!env || !env.node_id) return;
      get().updateStep({
        node_id: env.node_id,
        status: 'running',
      });
    });

    es.addEventListener('step_chunk', (e: MessageEvent) => {
      const env = parseEnvelope<{ chunk?: string }>(e.data);
      if (!env || !env.node_id) return;
      const chunk = typeof env.data?.chunk === 'string' ? env.data.chunk : '';
      if (!chunk) return;
      get().appendChunk(env.node_id, chunk);
    });

    es.addEventListener('step_completed', (e: MessageEvent) => {
      const env = parseEnvelope<{
        output?: unknown;
        output_file?: string | null;
        tokens_used?: number | null;
        cost_usd?: number | null;
        duration_ms?: number | null;
      }>(e.data);
      if (!env || !env.node_id) return;
      const payload = env.data ?? {};
      get().updateStep({
        node_id: env.node_id,
        status: 'completed',
        output: stringify(payload.output),
        // Backend emits "" when there's no artifact; normalise to
        // undefined so StepResult.output_file is a simple presence check.
        output_file:
          typeof payload.output_file === 'string' && payload.output_file !== ''
            ? payload.output_file
            : undefined,
        tokens_used: typeof payload.tokens_used === 'number' ? payload.tokens_used : undefined,
        cost: typeof payload.cost_usd === 'number' ? payload.cost_usd : undefined,
        duration_ms: typeof payload.duration_ms === 'number' ? payload.duration_ms : undefined,
        completed_at: env.timestamp,
      });
    });

    es.addEventListener('step_failed', (e: MessageEvent) => {
      const env = parseEnvelope<{ error?: string; failure?: unknown }>(e.data);
      if (!env || !env.node_id) return;
      const payloadErr = typeof env.data?.error === 'string' ? env.data.error : undefined;
      // The structured StepFailure rides in event.data.failure. Validate
      // it through coerceStepFailure so the UI never crashes on a
      // malformed payload — falls back to undefined and the legacy
      // `error` string still renders.
      const failure: StepFailure | undefined = coerceStepFailure(env.data?.failure);
      get().updateStep({
        node_id: env.node_id,
        status: 'failed',
        error: env.error ?? payloadErr,
        failure,
        completed_at: env.timestamp,
      });
    });

    es.addEventListener('execution_completed', (e: MessageEvent) => {
      const env = parseEnvelope<{ total_cost?: number; artifact_dir?: string }>(e.data);
      const total =
        env && typeof env.data?.total_cost === 'number'
          ? env.data.total_cost
          : get().totalCost;
      const dir =
        env && typeof env.data?.artifact_dir === 'string' && env.data.artifact_dir
          ? env.data.artifact_dir
          : null;
      set({
        status: 'completed',
        totalCost: total,
        isStreaming: false,
        artifactDir: dir,
      });
      get().disconnectSSE();
    });

    es.addEventListener('execution_failed', (e: MessageEvent) => {
      const env = parseEnvelope(e.data);
      void env; // currently unused; keep for future error surfacing
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
      artifactDir: null,
    });
  },
}));
