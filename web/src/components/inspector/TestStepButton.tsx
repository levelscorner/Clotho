import { useCallback, useState } from 'react';
import { usePipelineStore } from '../../stores/pipelineStore';
import { testNode, type NodeTestResult } from '../../lib/api';
import { coerceStepFailure } from '../../lib/failureSchema';
import { FailureDrawer } from '../execution/FailureDrawer';

interface TestStepButtonProps {
  nodeId: string;
}

/**
 * "Test step in isolation" inspector button. Pulls the live node from
 * pipelineStore (so unsaved config changes are tested too), POSTs to
 * /api/nodes/test, and renders the result inline.
 *
 * For agent nodes the prompt template usually carries `{{input}}`; we
 * pass nothing as upstream input so the user can validate the prompt
 * compiles and the credential resolves before plumbing real upstream
 * data. Future enhancement: a "Sample input" textarea above the button.
 */
export function TestStepButton({ nodeId }: TestStepButtonProps) {
  const node = usePipelineStore((s) => s.nodes.find((n) => n.id === nodeId));
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<NodeTestResult | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);

  const handleTest = useCallback(async () => {
    if (!node) return;
    setRunning(true);
    setResult(null);
    try {
      const r = await testNode({
        node: {
          id: node.id,
          type: node.data.nodeType,
          label: node.data.label,
          position: node.position,
          ports: node.data.ports,
          config: node.data.config,
        },
        inputs: {},
      });
      setResult(r);
    } catch (err: unknown) {
      setResult({
        duration_ms: 0,
        error: err instanceof Error ? err.message : 'Test request failed',
      });
    } finally {
      setRunning(false);
    }
  }, [node]);

  if (!node) return null;

  // Render the structured failure if present so we get the same drawer
  // experience as a real run. Coerce the wire payload defensively.
  const failure = result?.failure ? coerceStepFailure(result.failure) : undefined;
  const succeeded = !!result && !result.failure && !result.error;
  const previewText =
    typeof result?.output === 'string'
      ? result.output
      : result?.output != null
        ? JSON.stringify(result.output)
        : '';

  return (
    <div style={{ marginBottom: 12 }}>
      <button
        type="button"
        onClick={() => void handleTest()}
        disabled={running}
        style={{
          width: '100%',
          padding: '8px 12px',
          borderRadius: 'var(--radius-sm)',
          border: '1px solid var(--accent)',
          background: running ? 'var(--surface-overlay)' : 'var(--accent-soft)',
          color: 'var(--text-primary)',
          fontSize: 13,
          fontWeight: 600,
          cursor: running ? 'not-allowed' : 'pointer',
        }}
        title="Run this single node without saving the pipeline"
      >
        {running ? 'Testing…' : 'Test step in isolation'}
      </button>

      {result && (
        <div
          style={{
            marginTop: 8,
            padding: '8px 10px',
            borderRadius: 'var(--radius-sm)',
            border: '1px solid var(--surface-border)',
            background: 'var(--surface-overlay)',
            fontSize: 11,
            color: 'var(--text-secondary)',
          }}
        >
          <div
            style={{
              fontWeight: 600,
              color: succeeded ? '#22c55e' : '#f87171',
              marginBottom: 4,
            }}
          >
            {succeeded ? 'OK' : 'Failed'} · {result.duration_ms}ms
            {result.tokens_used != null && ` · ${result.tokens_used} tokens`}
            {result.cost_usd != null && ` · $${result.cost_usd.toFixed(5)}`}
          </div>
          {succeeded && previewText && (
            <div
              style={{
                fontFamily: 'var(--font-mono, ui-monospace, monospace)',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                color: 'var(--text-primary)',
                maxHeight: 120,
                overflowY: 'auto',
              }}
            >
              {previewText.slice(0, 400)}
              {previewText.length > 400 ? '…' : ''}
            </div>
          )}
          {failure && (
            <button
              type="button"
              onClick={() => setDrawerOpen(true)}
              style={{
                marginTop: 6,
                background: 'transparent',
                border: '1px solid var(--surface-border)',
                color: 'var(--text-primary)',
                padding: '4px 10px',
                borderRadius: 'var(--radius-sm)',
                fontSize: 11,
                cursor: 'pointer',
              }}
            >
              Why did it fail?
            </button>
          )}
          {!failure && result.error && (
            <div style={{ color: '#f87171' }}>{result.error}</div>
          )}
        </div>
      )}

      {drawerOpen && failure && (
        <FailureDrawer
          nodeId={nodeId}
          failure={failure}
          onClose={() => setDrawerOpen(false)}
        />
      )}
    </div>
  );
}
