import { useEffect, useState, useCallback } from 'react';
import { api, type ExecutionRow } from '../lib/api';
import { coerceStepFailure } from '../lib/failureSchema';
import { FailureDrawer } from '../components/execution/FailureDrawer';
import type { StepFailure } from '../lib/types';

// ---------------------------------------------------------------------------
// Status badge palette — mirrors STATUS_COLORS in ExecutionStatus.tsx so
// the executions list reads consistently with the top-bar.
// ---------------------------------------------------------------------------

const STATUS_COLORS: Record<string, string> = {
  pending: '#64748b',
  running: '#3b82f6',
  completed: '#22c55e',
  failed: '#ef4444',
  cancelled: '#f59e0b',
  skipped: '#6b7280',
};

const FILTERS = ['all', 'failed', 'completed', 'running'] as const;
type Filter = (typeof FILTERS)[number];

/**
 * Read-only paginated executions list. Supports filtering by status
 * and inline FailureDrawer opening for failed runs (so a user can
 * triage a failure without re-running). The retry button kicks off
 * a fresh execution against the same pipeline_version_id.
 *
 * Routed via pathname `/executions` — see App.tsx::isExecutionsRoute.
 */
export default function ExecutionsPage() {
  const [filter, setFilter] = useState<Filter>('failed');
  const [rows, setRows] = useState<ExecutionRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [drawer, setDrawer] = useState<{
    nodeId: string;
    failure: StepFailure;
  } | null>(null);

  const fetchRows = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.executions.list({
        status: filter === 'all' ? undefined : filter,
        limit: 50,
      });
      setRows(data);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load executions');
    } finally {
      setLoading(false);
    }
  }, [filter]);

  useEffect(() => {
    void fetchRows();
  }, [fetchRows]);

  const handleRetry = useCallback(
    async (id: string) => {
      try {
        await api.executions.retry(id);
        await fetchRows();
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : 'Retry failed');
      }
    },
    [fetchRows],
  );

  const handleViewFailure = useCallback((row: ExecutionRow) => {
    const failure = coerceStepFailure(row.failure);
    if (!failure) return;
    // node_id at the execution level is the FIRST failed step; the
    // backend persisted that on the executions row. We don't carry it
    // through here, so use a placeholder. The drawer's "Rerun from this
    // node" still works via executionId from useExecutionStore — but
    // that store isn't loaded here, so the button is a no-op on this
    // page. Acceptable for an MVP; a future iteration can plumb the
    // failed_node_id through the execution row.
    setDrawer({ nodeId: row.id, failure });
  }, []);

  return (
    <div
      style={{
        minHeight: '100vh',
        background: '#0f0f14',
        color: '#ececf0',
        fontFamily: "'Inter', sans-serif",
        padding: 24,
      }}
    >
      <header style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 20 }}>
        <a
          href="/"
          style={{ color: '#8888a0', textDecoration: 'none', fontSize: 13 }}
        >
          ← Back to canvas
        </a>
        <h1 style={{ fontSize: 18, fontWeight: 600, margin: 0 }}>Executions</h1>
        <div style={{ flex: 1 }} />
        <div style={{ display: 'flex', gap: 4 }}>
          {FILTERS.map((f) => (
            <button
              key={f}
              type="button"
              onClick={() => setFilter(f)}
              style={{
                padding: '4px 10px',
                fontSize: 12,
                borderRadius: 4,
                border: '1px solid #2e2e38',
                background: filter === f ? '#e5a84b' : 'transparent',
                color: filter === f ? '#121216' : '#ececf0',
                cursor: 'pointer',
                textTransform: 'capitalize',
              }}
            >
              {f}
            </button>
          ))}
        </div>
      </header>

      {error && (
        <div
          style={{
            padding: '8px 12px',
            borderRadius: 6,
            background: 'rgba(248,113,113,0.12)',
            border: '1px solid rgba(248,113,113,0.25)',
            color: '#f87171',
            fontSize: 12,
            marginBottom: 12,
          }}
        >
          {error}
        </div>
      )}

      {loading ? (
        <div style={{ color: '#55556a', padding: '24px 0', fontSize: 13 }}>
          Loading executions…
        </div>
      ) : rows.length === 0 ? (
        <div
          style={{
            padding: '40px 16px',
            background: '#1a1a20',
            borderRadius: 8,
            border: '1px solid #2e2e38',
            textAlign: 'center',
            color: '#8888a0',
          }}
        >
          No {filter === 'all' ? '' : filter} executions yet.
        </div>
      ) : (
        <table
          style={{
            width: '100%',
            borderCollapse: 'collapse',
            background: '#1a1a20',
            borderRadius: 8,
            overflow: 'hidden',
          }}
        >
          <thead>
            <tr style={{ background: '#222228' }}>
              {['Status', 'ID', 'Started', 'Duration', 'Cost', 'Tokens', 'Actions'].map((h) => (
                <th
                  key={h}
                  style={{
                    padding: '10px 12px',
                    textAlign: 'left',
                    fontSize: 11,
                    fontWeight: 600,
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                    color: '#8888a0',
                  }}
                >
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => {
              const duration =
                row.completed_at && row.started_at
                  ? new Date(row.completed_at).getTime() -
                    new Date(row.started_at).getTime()
                  : null;
              const failure = coerceStepFailure(row.failure);
              return (
                <tr
                  key={row.id}
                  style={{ borderTop: '1px solid #2e2e38' }}
                >
                  <td style={{ padding: '10px 12px' }}>
                    <span
                      style={{
                        display: 'inline-block',
                        padding: '2px 8px',
                        borderRadius: 4,
                        fontSize: 10,
                        fontWeight: 700,
                        background: STATUS_COLORS[row.status] ?? '#475569',
                        color: '#fff',
                        textTransform: 'uppercase',
                      }}
                    >
                      {row.status}
                    </span>
                  </td>
                  <td
                    style={{
                      padding: '10px 12px',
                      fontFamily: 'ui-monospace, monospace',
                      fontSize: 11,
                      color: '#8888a0',
                    }}
                  >
                    {row.id.slice(0, 8)}
                  </td>
                  <td style={{ padding: '10px 12px', fontSize: 12 }}>
                    {row.started_at
                      ? new Date(row.started_at).toLocaleString()
                      : '—'}
                  </td>
                  <td style={{ padding: '10px 12px', fontSize: 12 }}>
                    {duration != null ? `${(duration / 1000).toFixed(1)}s` : '—'}
                  </td>
                  <td style={{ padding: '10px 12px', fontSize: 12 }}>
                    {row.total_cost != null ? `$${row.total_cost.toFixed(4)}` : '—'}
                  </td>
                  <td style={{ padding: '10px 12px', fontSize: 12 }}>
                    {row.total_tokens ?? '—'}
                  </td>
                  <td style={{ padding: '10px 12px', display: 'flex', gap: 6 }}>
                    {failure && (
                      <button
                        type="button"
                        onClick={() => handleViewFailure(row)}
                        style={{
                          padding: '4px 8px',
                          fontSize: 11,
                          borderRadius: 4,
                          border: '1px solid #f87171',
                          background: 'transparent',
                          color: '#f87171',
                          cursor: 'pointer',
                        }}
                      >
                        Why?
                      </button>
                    )}
                    <button
                      type="button"
                      onClick={() => void handleRetry(row.id)}
                      style={{
                        padding: '4px 8px',
                        fontSize: 11,
                        borderRadius: 4,
                        border: '1px solid #2e2e38',
                        background: 'transparent',
                        color: '#ececf0',
                        cursor: 'pointer',
                      }}
                    >
                      Retry
                    </button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}

      {drawer && (
        <FailureDrawer
          nodeId={drawer.nodeId}
          failure={drawer.failure}
          onClose={() => setDrawer(null)}
        />
      )}
    </div>
  );
}
