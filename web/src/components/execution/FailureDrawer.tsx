import { useState } from 'react';
import type { FailureClass, StepFailure, StepResult } from '../../lib/types';
import { useExecutionStore } from '../../stores/executionStore';

// Class → (background, foreground) tuple for the badge. Color choice:
//   - amber for transient/transient-ish failures (rate_limit, timeout)
//   - red for hard stops (auth, internal, circuit_open)
//   - purple for shape/validation problems (need user attention but
//     usually not credential-related)
//   - gray for network blips
const CLASS_COLORS: Record<FailureClass, { bg: string; fg: string; label: string }> = {
  network:        { bg: '#475569', fg: '#f1f5f9', label: 'Network' },
  rate_limit:     { bg: '#b45309', fg: '#fef3c7', label: 'Rate limit' },
  timeout:        { bg: '#b45309', fg: '#fef3c7', label: 'Timeout' },
  auth:           { bg: '#b91c1c', fg: '#fee2e2', label: 'Auth' },
  provider_5xx:   { bg: '#b45309', fg: '#fef3c7', label: 'Provider 5xx' },
  provider_4xx:   { bg: '#7c3aed', fg: '#ede9fe', label: 'Provider 4xx' },
  validation:     { bg: '#7c3aed', fg: '#ede9fe', label: 'Validation' },
  output_shape:   { bg: '#7c3aed', fg: '#ede9fe', label: 'Output shape' },
  output_quality: { bg: '#7c3aed', fg: '#ede9fe', label: 'Output quality' },
  cost_cap:       { bg: '#b45309', fg: '#fef3c7', label: 'Cost cap' },
  circuit_open:   { bg: '#991b1b', fg: '#fee2e2', label: 'Circuit open' },
  internal:       { bg: '#b91c1c', fg: '#fee2e2', label: 'Internal' },
};

interface FailureDrawerProps {
  /** Node the failure belongs to. Used as drawer title + retry target. */
  nodeId: string;
  failure: StepFailure;
  /** Closes the drawer. */
  onClose: () => void;
}

/**
 * Modal drawer surfacing the structured StepFailure for a single node.
 * Shows the class badge, headline message, scrubbed cause, and a hint
 * that often points at a config fix the user can do right now. Two
 * actions: "Rerun from this node" (uses existing engine.RerunFromNode)
 * and "Copy diagnostic" (full StepFailure JSON for bug reports).
 */
export function FailureDrawer({ nodeId, failure, onClose }: FailureDrawerProps) {
  const executionId = useExecutionStore((s) => s.executionId);
  const retryNode = useExecutionStore((s) => s.retryNode);
  const [copied, setCopied] = useState(false);
  const [causeOpen, setCauseOpen] = useState(false);

  const palette = CLASS_COLORS[failure.class];

  const handleRetry = () => {
    if (!executionId) return;
    retryNode(executionId, nodeId);
    onClose();
  };

  const handleCopy = async () => {
    const diag = {
      execution_id: executionId,
      node_id: nodeId,
      failure,
    };
    try {
      await navigator.clipboard.writeText(JSON.stringify(diag, null, 2));
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // Fallback: ignore. Older browsers without clipboard API just see
      // the button click do nothing.
    }
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Step failure details"
      onClick={onClose}
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.5)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 9999,
      }}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          background: 'var(--surface-base)',
          color: 'var(--text-primary)',
          width: 'min(560px, 92vw)',
          maxHeight: '80vh',
          overflowY: 'auto',
          borderRadius: 'var(--radius-md)',
          border: '1px solid var(--surface-border)',
          padding: 20,
          boxShadow: '0 20px 60px rgba(0,0,0,0.4)',
        }}
      >
        <header
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            marginBottom: 12,
          }}
        >
          <span
            style={{
              padding: '3px 10px',
              borderRadius: 999,
              background: palette.bg,
              color: palette.fg,
              fontSize: 11,
              fontWeight: 700,
              textTransform: 'uppercase',
              letterSpacing: '0.04em',
            }}
          >
            {palette.label}
          </span>
          {failure.provider && (
            <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>
              {failure.provider}
              {failure.model ? ` · ${failure.model}` : ''}
            </span>
          )}
          <span style={{ flex: 1 }} />
          {failure.attempts > 0 && (
            <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
              attempt {failure.attempts}
            </span>
          )}
          <button
            type="button"
            aria-label="Close"
            onClick={onClose}
            style={{
              background: 'transparent',
              border: '1px solid var(--surface-border)',
              color: 'var(--text-secondary)',
              borderRadius: 'var(--radius-sm)',
              width: 28,
              height: 28,
              cursor: 'pointer',
              fontSize: 14,
            }}
          >
            {'\u2715'}
          </button>
        </header>

        <h2
          style={{
            margin: '4px 0 8px',
            fontSize: 16,
            fontWeight: 600,
            color: 'var(--text-primary)',
          }}
        >
          {failure.message}
        </h2>

        {failure.hint && (
          <div
            style={{
              padding: '10px 12px',
              borderRadius: 'var(--radius-sm)',
              background: 'var(--accent-soft)',
              color: 'var(--text-primary)',
              fontSize: 13,
              lineHeight: 1.45,
              marginBottom: 14,
            }}
          >
            <strong style={{ marginRight: 6 }}>Suggestion:</strong>
            {failure.hint}
          </div>
        )}

        {failure.cause && (
          <details
            open={causeOpen}
            onToggle={(e) => setCauseOpen((e.target as HTMLDetailsElement).open)}
            style={{ marginBottom: 14 }}
          >
            <summary
              style={{
                cursor: 'pointer',
                fontSize: 12,
                color: 'var(--text-muted)',
                marginBottom: 6,
              }}
            >
              Underlying cause
            </summary>
            <pre
              style={{
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                fontFamily: 'var(--font-mono, ui-monospace, monospace)',
                fontSize: 11,
                lineHeight: 1.5,
                background: 'var(--surface-overlay)',
                padding: '10px 12px',
                borderRadius: 'var(--radius-sm)',
                margin: 0,
                color: 'var(--text-secondary)',
              }}
            >
              {failure.cause}
            </pre>
          </details>
        )}

        <div
          style={{
            display: 'flex',
            gap: 8,
            marginTop: 16,
            paddingTop: 14,
            borderTop: '1px solid var(--surface-border)',
          }}
        >
          {failure.retryable && (
            <button
              type="button"
              onClick={handleRetry}
              style={{
                background: 'var(--accent)',
                color: '#fff',
                border: 'none',
                borderRadius: 'var(--radius-sm)',
                padding: '8px 14px',
                fontSize: 13,
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              Rerun from this node
            </button>
          )}
          <button
            type="button"
            onClick={handleCopy}
            style={{
              background: 'transparent',
              color: 'var(--text-primary)',
              border: '1px solid var(--surface-border)',
              borderRadius: 'var(--radius-sm)',
              padding: '8px 14px',
              fontSize: 13,
              cursor: 'pointer',
            }}
          >
            {copied ? 'Copied' : 'Copy diagnostic'}
          </button>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Helper: inline tooltip text for a failed node (used by AgentNode/ToolNode
// hover overlay). Returns the headline class + provider so users get
// immediate feedback without opening the drawer.
// ---------------------------------------------------------------------------

export function failureTooltipText(failure: StepFailure): string {
  const parts: string[] = [CLASS_COLORS[failure.class]?.label ?? failure.class];
  if (failure.provider) parts.push(failure.provider);
  if (failure.attempts > 0) parts.push(`attempt ${failure.attempts}`);
  return parts.join(' · ');
}

// ---------------------------------------------------------------------------
// Helper: pick the StepFailure to surface in the top-bar — first failed
// step in iteration order. Used by ExecutionStatus to mount the drawer.
// ---------------------------------------------------------------------------

export function firstFailureFromMap(
  stepResults: ReadonlyMap<string, StepResult>,
): { nodeId: string; failure: StepFailure } | null {
  for (const [nodeId, sr] of stepResults) {
    if (sr.status === 'failed' && sr.failure) {
      return { nodeId, failure: sr.failure };
    }
  }
  return null;
}
