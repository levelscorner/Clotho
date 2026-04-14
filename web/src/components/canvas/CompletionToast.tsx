import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useExecutionStore } from '../../stores/executionStore';
import { usePipelineStore } from '../../stores/pipelineStore';
import './CompletionToast.css';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const AUTO_DISMISS_MS = 4000;
const CASCADE_STEP_MS = 200;
const CASCADE_TOTAL_MS = 1000;
const PULSE_CLASS = 'clotho-node--completion-pulse';

// ---------------------------------------------------------------------------
// Formatters
// ---------------------------------------------------------------------------

function formatDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) return '—';
  if (ms < 1000) return `${Math.round(ms)}ms`;
  const seconds = ms / 1000;
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const minutes = Math.floor(seconds / 60);
  const rem = Math.round(seconds - minutes * 60);
  return `${minutes}m ${rem}s`;
}

function formatCost(usd: number): string {
  if (!Number.isFinite(usd) || usd <= 0) return '$0.00';
  if (usd < 0.01) return `$${usd.toFixed(4)}`;
  return `$${usd.toFixed(2)}`;
}

function formatTokens(tokens: number): string {
  if (!Number.isFinite(tokens) || tokens <= 0) return '—';
  if (tokens >= 1000) return `${(tokens / 1000).toFixed(1)}k`;
  return `${tokens}`;
}

// ---------------------------------------------------------------------------
// Cascade pulse — class injection with per-node delay, respects reduced motion.
// ---------------------------------------------------------------------------

function reducedMotion(): boolean {
  if (typeof window === 'undefined' || !window.matchMedia) return false;
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

function triggerCascade(orderedNodeIds: ReadonlyArray<string>): () => void {
  if (typeof document === 'undefined') return () => {};
  if (reducedMotion()) return () => {};

  const applied: { el: Element; delay: string | null }[] = [];

  orderedNodeIds.forEach((id, index) => {
    // React Flow wraps each node in a [data-id="<id>"] container.
    const wrapper = document.querySelector(`[data-id="${CSS.escape(id)}"]`);
    const inner = wrapper?.querySelector('.clotho-node') ?? wrapper;
    if (!(inner instanceof HTMLElement)) return;
    const prev = inner.style.getPropertyValue('--pulse-delay');
    inner.style.setProperty('--pulse-delay', `${index * CASCADE_STEP_MS}ms`);
    inner.classList.add(PULSE_CLASS);
    applied.push({ el: inner, delay: prev || null });
  });

  const timeoutId = window.setTimeout(() => {
    applied.forEach(({ el, delay }) => {
      el.classList.remove(PULSE_CLASS);
      if (el instanceof HTMLElement) {
        if (delay === null) {
          el.style.removeProperty('--pulse-delay');
        } else {
          el.style.setProperty('--pulse-delay', delay);
        }
      }
    });
  }, CASCADE_TOTAL_MS + 100);

  return () => {
    window.clearTimeout(timeoutId);
    applied.forEach(({ el, delay }) => {
      el.classList.remove(PULSE_CLASS);
      if (el instanceof HTMLElement) {
        if (delay === null) {
          el.style.removeProperty('--pulse-delay');
        } else {
          el.style.setProperty('--pulse-delay', delay);
        }
      }
    });
  };
}

// ---------------------------------------------------------------------------
// Topological order (left-to-right, breaking ties by x-position)
// ---------------------------------------------------------------------------

interface GraphNode {
  id: string;
  x: number;
}

interface GraphEdge {
  source: string;
  target: string;
}

function topoOrder(nodes: GraphNode[], edges: GraphEdge[]): string[] {
  const inDegree = new Map<string, number>();
  const adj = new Map<string, string[]>();
  for (const n of nodes) inDegree.set(n.id, 0);
  for (const e of edges) {
    inDegree.set(e.target, (inDegree.get(e.target) ?? 0) + 1);
    const list = adj.get(e.source) ?? [];
    list.push(e.target);
    adj.set(e.source, list);
  }
  const nodeById = new Map(nodes.map((n) => [n.id, n]));
  const ready: string[] = [];
  inDegree.forEach((deg, id) => {
    if (deg === 0) ready.push(id);
  });
  ready.sort((a, b) => (nodeById.get(a)?.x ?? 0) - (nodeById.get(b)?.x ?? 0));

  const out: string[] = [];
  while (ready.length > 0) {
    const id = ready.shift() as string;
    out.push(id);
    const targets = adj.get(id) ?? [];
    const newly: string[] = [];
    for (const t of targets) {
      const deg = (inDegree.get(t) ?? 0) - 1;
      inDegree.set(t, deg);
      if (deg === 0) newly.push(t);
    }
    newly.sort((a, b) => (nodeById.get(a)?.x ?? 0) - (nodeById.get(b)?.x ?? 0));
    ready.push(...newly);
  }
  // Fallback: append any nodes left out due to cycles.
  for (const n of nodes) if (!out.includes(n.id)) out.push(n.id);
  return out;
}

// ---------------------------------------------------------------------------
// Sink node (final output) detection: lowest out-degree, highest topo index.
// ---------------------------------------------------------------------------

function findSinkNodeId(
  order: ReadonlyArray<string>,
  edges: ReadonlyArray<GraphEdge>,
): string | null {
  if (order.length === 0) return null;
  const outDegree = new Map<string, number>();
  for (const id of order) outDegree.set(id, 0);
  for (const e of edges) {
    outDegree.set(e.source, (outDegree.get(e.source) ?? 0) + 1);
  }
  // Walk topo order from the end, return the first node with out-degree 0.
  for (let i = order.length - 1; i >= 0; i -= 1) {
    if ((outDegree.get(order[i]) ?? 0) === 0) return order[i];
  }
  return order[order.length - 1];
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function CompletionToast(): JSX.Element | null {
  const status = useExecutionStore((s) => s.status);
  const stepResults = useExecutionStore((s) => s.stepResults);
  const totalCost = useExecutionStore((s) => s.totalCost);
  const nodes = usePipelineStore((s) => s.nodes);
  const edges = usePipelineStore((s) => s.edges);
  const setSelectedNode = usePipelineStore((s) => s.setSelectedNode);

  const [visible, setVisible] = useState(false);
  const prevStatusRef = useRef<typeof status>(null);
  const cascadeCleanupRef = useRef<(() => void) | null>(null);

  // Aggregates
  const { totalTokens, totalDurationMs, hasFailure } = useMemo(() => {
    let tokens = 0;
    let duration = 0;
    let failed = false;
    stepResults.forEach((r) => {
      tokens += r.tokens_used ?? 0;
      duration += r.duration_ms ?? 0;
      if (r.status === 'failed') failed = true;
    });
    return { totalTokens: tokens, totalDurationMs: duration, hasFailure: failed };
  }, [stepResults]);

  // Transition detection: running → completed with no failures.
  useEffect(() => {
    const prev = prevStatusRef.current;
    prevStatusRef.current = status;

    if (status !== 'completed') return;
    if (prev === 'completed') return;
    if (hasFailure) return;

    setVisible(true);

    // Cascade pulse across the canvas in topological order.
    const order = topoOrder(
      nodes.map((n) => ({ id: n.id, x: n.position.x })),
      edges.map((e) => ({ source: e.source, target: e.target })),
    );
    cascadeCleanupRef.current?.();
    cascadeCleanupRef.current = triggerCascade(order);

    const timeoutId = window.setTimeout(() => {
      setVisible(false);
    }, AUTO_DISMISS_MS);

    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [status, hasFailure, nodes, edges]);

  // Cleanup cascade on unmount.
  useEffect(() => {
    return () => {
      cascadeCleanupRef.current?.();
    };
  }, []);

  const onViewOutput = useCallback(() => {
    const order = topoOrder(
      nodes.map((n) => ({ id: n.id, x: n.position.x })),
      edges.map((e) => ({ source: e.source, target: e.target })),
    );
    const sinkId = findSinkNodeId(
      order,
      edges.map((e) => ({ source: e.source, target: e.target })),
    );
    if (sinkId) setSelectedNode(sinkId);
    setVisible(false);
  }, [nodes, edges, setSelectedNode]);

  if (!visible) return null;

  return (
    <div
      className="completion-toast"
      role="status"
      aria-live="polite"
      aria-label="Pipeline completed"
    >
      <div className="completion-toast__grid">
        <div className="completion-toast__stat">
          <span className="completion-toast__label">Time</span>
          <span className="completion-toast__value">
            {formatDuration(totalDurationMs)}
          </span>
        </div>
        <div className="completion-toast__stat">
          <span className="completion-toast__label">Cost</span>
          <span className="completion-toast__value">{formatCost(totalCost)}</span>
        </div>
        <div className="completion-toast__stat">
          <span className="completion-toast__label">Tokens</span>
          <span className="completion-toast__value">
            {formatTokens(totalTokens)}
          </span>
        </div>
      </div>
      <button
        type="button"
        className="completion-toast__cta"
        onClick={onViewOutput}
      >
        VIEW OUTPUT →
      </button>
    </div>
  );
}
