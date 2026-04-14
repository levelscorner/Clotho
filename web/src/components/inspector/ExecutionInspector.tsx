import type { StepResult, ExecutionStatus } from '../../lib/types';
import { InspectorGroup } from './InspectorGroup';

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
  const hasDetails = Boolean(step.output || step.error);

  return (
    <div style={{ fontSize: 13 }}>
      <InspectorGroup title="Summary" defaultOpen>
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
      </InspectorGroup>

      {hasDetails && (
        <InspectorGroup title="Details" forceOpen={Boolean(step.error)}>
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
              <OutputPreview output={step.output} />
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
        </InspectorGroup>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Output preview — renders the step output with media-type awareness.
//
// Image/audio/video data URIs become inline <img>/<audio>/<video> elements
// so the inspector shows a real preview instead of a multi-megabyte base64
// string. Everything else falls back to a scrollable <pre>.
// ---------------------------------------------------------------------------

function OutputPreview({ output }: { output: string }) {
  const mediaBlock: React.CSSProperties = {
    background: '#1a1c2e',
    padding: 8,
    borderRadius: 4,
    border: '1px solid #334155',
    maxHeight: 320,
  };

  if (output.startsWith('data:image/')) {
    return (
      <div style={mediaBlock}>
        <img
          src={output}
          alt="Generated output"
          style={{ width: '100%', height: 'auto', borderRadius: 3, display: 'block' }}
        />
      </div>
    );
  }
  if (output.startsWith('data:audio/')) {
    return (
      <div style={mediaBlock}>
        <audio controls src={output} style={{ width: '100%' }} />
      </div>
    );
  }
  if (output.startsWith('data:video/')) {
    return (
      <div style={mediaBlock}>
        <video controls src={output} style={{ width: '100%', borderRadius: 3 }} />
      </div>
    );
  }
  // Truncate very long plain-text outputs so we never paint a 1MB <pre>.
  const MAX = 8000;
  const shown = output.length > MAX ? output.slice(0, MAX) + '\n…(truncated)' : output;
  return (
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
      {shown}
    </pre>
  );
}
