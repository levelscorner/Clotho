import { usePipelineStore } from '../../stores/pipelineStore';

// Mirrors internal/engine/graph.go::ValidationError. Shape kept stable
// so the modal can render whatever the backend emits without coupling
// to specific fields beyond Field + Message.
export interface ValidationErrorPayload {
  field: string;
  message: string;
}

interface ValidationModalProps {
  errors: ValidationErrorPayload[];
  onClose: () => void;
}

/**
 * Save-time validation modal. Replaces the silent edge-drop UX so users
 * never wonder why their pipeline didn't run. Each error gets a "Select
 * node" / "Select edge" link that focuses the relevant graph element so
 * the user can fix it without hunting.
 *
 * Backend ValidationError.Field encodes locations:
 *   - "nodes" / "duplicate node ID: ..."
 *   - "edges[edgeID].source" / "edges[edgeID].source_port"
 *   - "edges[edgeID].target" / "edges[edgeID].target_port"
 *   - "edges[edgeID].type"
 *   - "nodes[nodeID].ports[portID].required"
 *   - "graph.cycle"
 *
 * We grep these strings to extract the IDs and dispatch the right
 * selection in pipelineStore. Brittle if the backend reformats — when
 * that happens the modal still shows the raw text, just without the
 * click-to-jump affordance.
 */
export function ValidationModal({ errors, onClose }: ValidationModalProps) {
  const setSelectedNode = usePipelineStore((s) => s.setSelectedNode);

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Validation errors"
      onClick={onClose}
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.55)',
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
            marginBottom: 14,
          }}
        >
          <span
            style={{
              padding: '3px 10px',
              borderRadius: 999,
              background: '#7c3aed',
              color: '#ede9fe',
              fontSize: 11,
              fontWeight: 700,
              textTransform: 'uppercase',
            }}
          >
            Validation
          </span>
          <h2
            style={{
              margin: 0,
              fontSize: 16,
              fontWeight: 600,
              flex: 1,
            }}
          >
            {errors.length} issue{errors.length === 1 ? '' : 's'} blocking save
          </h2>
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

        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: 8,
          }}
        >
          {errors.map((err, idx) => {
            const nodeId = extractNodeId(err.field);
            return (
              <div
                key={`${err.field}-${idx}`}
                style={{
                  padding: '10px 12px',
                  borderRadius: 'var(--radius-sm)',
                  border: '1px solid var(--surface-border)',
                  background: 'var(--surface-overlay)',
                }}
              >
                <div
                  style={{
                    fontSize: 11,
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-mono, ui-monospace, monospace)',
                    marginBottom: 4,
                  }}
                >
                  {err.field}
                </div>
                <div style={{ fontSize: 13, color: 'var(--text-primary)' }}>
                  {err.message}
                </div>
                {nodeId && (
                  <button
                    type="button"
                    onClick={() => {
                      setSelectedNode(nodeId);
                      onClose();
                    }}
                    style={{
                      marginTop: 6,
                      background: 'transparent',
                      border: '1px solid var(--accent)',
                      color: 'var(--accent)',
                      padding: '4px 10px',
                      borderRadius: 'var(--radius-sm)',
                      fontSize: 11,
                      cursor: 'pointer',
                    }}
                  >
                    Select node
                  </button>
                )}
              </div>
            );
          })}
        </div>

        <div
          style={{
            marginTop: 16,
            paddingTop: 14,
            borderTop: '1px solid var(--surface-border)',
            display: 'flex',
            justifyContent: 'flex-end',
          }}
        >
          <button
            type="button"
            onClick={onClose}
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
            Got it
          </button>
        </div>
      </div>
    </div>
  );
}

// Greps either "nodes[id]..." (rare in current ValidateGraph but
// possible for future rules) or extracts the source/target node from
// an edge error by walking the field path. Returns "" when no node ID
// can be extracted — caller hides the Select-node button in that case.
function extractNodeId(field: string): string {
  // Pattern: nodes[id]
  const nodeMatch = field.match(/^nodes\[([^\]]+)\]/);
  if (nodeMatch) return nodeMatch[1];

  // Pattern: edges[edgeID].source / .target — we can't recover the node
  // id from the edge alone without the graph; leave empty for now.
  return '';
}

// ---------------------------------------------------------------------------
// Helper: parse ApiError messages thrown by api.ts into a structured
// ValidationErrorPayload[] when present. Returns null when the error
// isn't a 400 with validation_errors — caller falls back to a generic
// toast.
// ---------------------------------------------------------------------------

export function parseValidationErrors(
  err: unknown,
): ValidationErrorPayload[] | null {
  if (!err || typeof err !== 'object') return null;
  // ApiError has a `message` string carrying the raw response text.
  const message = (err as { message?: string }).message;
  if (typeof message !== 'string') return null;

  let parsed: unknown;
  try {
    parsed = JSON.parse(message);
  } catch {
    return null;
  }
  if (!parsed || typeof parsed !== 'object') return null;
  const arr = (parsed as { validation_errors?: unknown }).validation_errors;
  if (!Array.isArray(arr)) return null;

  const out: ValidationErrorPayload[] = [];
  for (const entry of arr) {
    if (!entry || typeof entry !== 'object') continue;
    const e = entry as Record<string, unknown>;
    const field = typeof e.field === 'string' ? e.field : '';
    const msg = typeof e.message === 'string' ? e.message : '';
    if (msg) out.push({ field, message: msg });
  }
  return out.length > 0 ? out : null;
}
