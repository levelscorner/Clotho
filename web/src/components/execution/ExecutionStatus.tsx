import { usePipelineStore } from '../../stores/pipelineStore';
import { useExecutionStore } from '../../stores/executionStore';
import type { ExecutionStatus as ExecStatus } from '../../lib/types';

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

  if (!status) return null;

  const completedCount = Array.from(stepResults.values()).filter(
    (r) => r.status === 'completed' || r.status === 'failed',
  ).length;

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
    </div>
  );
}
