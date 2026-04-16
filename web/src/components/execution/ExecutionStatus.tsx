import { useState } from 'react';
import { usePipelineStore } from '../../stores/pipelineStore';
import { useExecutionStore } from '../../stores/executionStore';
import type { ExecutionStatus as ExecStatus } from '../../lib/types';
import { FailureDrawer, firstFailureFromMap } from './FailureDrawer';

// ---------------------------------------------------------------------------
// Status badge colour
// ---------------------------------------------------------------------------

const STATUS_COLORS: Record<ExecStatus, string> = {
  pending: '#64748b',
  running: '#3b82f6',
  completed: '#22c55e',
  failed: '#ef4444',
  cancelled: '#f59e0b',
  skipped: '#6b7280',
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function ExecutionStatus() {
  const status = useExecutionStore((s) => s.status);
  const totalCost = useExecutionStore((s) => s.totalCost);
  const stepResults = useExecutionStore((s) => s.stepResults);
  const totalNodes = usePipelineStore((s) => s.nodes.length);
  const [drawerOpen, setDrawerOpen] = useState(false);

  if (!status) return null;

  const completedCount = Array.from(stepResults.values()).filter(
    (r) => r.status === 'completed' || r.status === 'failed',
  ).length;
  const failureCount = Array.from(stepResults.values()).filter(
    (r) => r.status === 'failed',
  ).length;
  const surfaced = firstFailureFromMap(stepResults);

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        fontSize: 12,
        color: '#94a3b8',
      }}
    >
      <span
        style={{
          display: 'inline-block',
          padding: '2px 8px',
          borderRadius: 4,
          fontSize: 11,
          fontWeight: 600,
          background: STATUS_COLORS[status],
          color: '#fff',
        }}
      >
        {status}
      </span>

      {totalNodes > 0 && (
        <span>
          {completedCount}/{totalNodes} nodes
        </span>
      )}

      {totalCost > 0 && <span>Cost: ${totalCost.toFixed(4)}</span>}

      {failureCount > 0 && surfaced && (
        <button
          type="button"
          onClick={() => setDrawerOpen(true)}
          aria-label={`Open failure details (${failureCount})`}
          style={{
            marginLeft: 'auto',
            background: 'transparent',
            border: '1px solid #ef4444',
            color: '#ef4444',
            borderRadius: 4,
            padding: '2px 8px',
            fontSize: 11,
            fontWeight: 600,
            cursor: 'pointer',
          }}
        >
          {failureCount} failure{failureCount > 1 ? 's' : ''} — why?
        </button>
      )}

      {drawerOpen && surfaced && (
        <FailureDrawer
          nodeId={surfaced.nodeId}
          failure={surfaced.failure}
          onClose={() => setDrawerOpen(false)}
        />
      )}
    </div>
  );
}
