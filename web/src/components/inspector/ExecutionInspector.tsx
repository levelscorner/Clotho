import type { StepResult, ExecutionStatus } from '../../lib/types';

// ---------------------------------------------------------------------------
// Status badge colours
// ---------------------------------------------------------------------------

const STATUS_COLORS: Record<ExecutionStatus, string> = {
  pending: '#64748b',
  running: '#3b82f6',
  completed: '#22c55e',
  failed: '#ef4444',
  cancelled: '#f59e0b',
  skipped: '#6b7280',
};

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface ExecutionInspectorProps {
  step: StepResult;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function ExecutionInspector({ step }: ExecutionInspectorProps) {
  return (
    <div style={{ fontSize: 13 }}>
      <div style={{ marginBottom: 10, display: 'flex', alignItems: 'center', gap: 8 }}>
        <span
          style={{
            display: 'inline-block',
            padding: '2px 8px',
            borderRadius: 4,
            fontSize: 11,
            fontWeight: 600,
            background: STATUS_COLORS[step.status],
            color: '#fff',
          }}
        >
          {step.status}
        </span>
      </div>

      {step.output && (
        <div style={{ marginBottom: 10 }}>
          <div
            style={{
              fontSize: 11,
              fontWeight: 600,
              color: '#64748b',
              marginBottom: 4,
              textTransform: 'uppercase',
            }}
          >
            Output
          </div>
          <pre
            style={{
              background: '#1a1c2e',
              padding: 8,
              borderRadius: 4,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
              maxHeight: 300,
              overflow: 'auto',
              fontSize: 12,
              color: '#e2e8f0',
              border: '1px solid #334155',
            }}
          >
            {step.output}
          </pre>
        </div>
      )}

      {step.error && (
        <div style={{ marginBottom: 10 }}>
          <div
            style={{
              fontSize: 11,
              fontWeight: 600,
              color: '#ef4444',
              marginBottom: 4,
              textTransform: 'uppercase',
            }}
          >
            Error
          </div>
          <pre
            style={{
              background: '#1a1c2e',
              padding: 8,
              borderRadius: 4,
              whiteSpace: 'pre-wrap',
              color: '#fca5a5',
              fontSize: 12,
              border: '1px solid #7f1d1d',
            }}
          >
            {step.error}
          </pre>
        </div>
      )}

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          gap: 8,
          fontSize: 12,
          color: '#94a3b8',
        }}
      >
        {step.tokens_used != null && (
          <div>Tokens: {step.tokens_used.toLocaleString()}</div>
        )}
        {step.cost != null && <div>Cost: ${step.cost.toFixed(4)}</div>}
        {step.duration_ms != null && (
          <div>Duration: {(step.duration_ms / 1000).toFixed(1)}s</div>
        )}
      </div>
    </div>
  );
}
