import { useCallback, useEffect, useRef, useState } from 'react';
import { usePipelineStore } from '../../stores/pipelineStore';
import { useExecutionStore } from '../../stores/executionStore';
import { useUIStore } from '../../stores/uiStore';
import type { AgentNodeConfig, ToolNodeConfig, MediaNodeConfig } from '../../lib/types';
import { AgentInspector } from './AgentInspector';
import { ToolInspector } from './ToolInspector';
import { MediaInspector } from './MediaInspector';
import { ExecutionInspector } from './ExecutionInspector';

// ---------------------------------------------------------------------------
// NodeInspector — desktop right panel, tablet drawer, phone bottom-sheet.
//
// Layout is fully CSS-driven (see AppShell.css + responsive.css). JS-side we
// only add the overlay close affordance, focus trap, Escape handling, and
// backdrop rendering when the viewport is narrow.
// ---------------------------------------------------------------------------

const OVERLAY_QUERY = '(max-width: 1023px)';

function useIsOverlayMode(): boolean {
  const [matches, setMatches] = useState<boolean>(() =>
    typeof window !== 'undefined'
      ? window.matchMedia(OVERLAY_QUERY).matches
      : false,
  );

  useEffect(() => {
    if (typeof window === 'undefined') return;
    const mq = window.matchMedia(OVERLAY_QUERY);
    const onChange = (e: MediaQueryListEvent) => setMatches(e.matches);
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  }, []);

  return matches;
}

export function NodeInspector() {
  const selectedNodeId = usePipelineStore((s) => s.selectedNodeId);
  const nodes = usePipelineStore((s) => s.nodes);
  const removeNodes = usePipelineStore((s) => s.removeNodes);
  const setSelectedNode = usePipelineStore((s) => s.setSelectedNode);
  const executionStatus = useExecutionStore((s) => s.status);
  const stepResults = useExecutionStore((s) => s.stepResults);
  const dismissed = useUIStore((s) => s.mobileInspectorDismissed);
  const dismiss = useUIStore((s) => s.dismissMobileInspector);
  const resetDismiss = useUIStore((s) => s.resetMobileInspectorDismissed);

  const isOverlay = useIsOverlayMode();
  const closeBtnRef = useRef<HTMLButtonElement | null>(null);

  // Reset the dismissed flag whenever a new node is selected so the drawer
  // re-opens for the new selection.
  useEffect(() => {
    resetDismiss();
  }, [selectedNodeId, resetDismiss]);

  const handleDelete = useCallback(() => {
    if (selectedNodeId) {
      removeNodes([selectedNodeId]);
      setSelectedNode(null);
    }
  }, [selectedNodeId, removeNodes, setSelectedNode]);

  const handleClose = useCallback(() => {
    setSelectedNode(null);
    dismiss();
  }, [setSelectedNode, dismiss]);

  // Escape closes the overlay inspector (only while it's shown).
  useEffect(() => {
    if (!isOverlay || !selectedNodeId || dismissed) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        handleClose();
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [isOverlay, selectedNodeId, dismissed, handleClose]);

  // Simple focus trap — focus the close button when overlay opens.
  useEffect(() => {
    if (isOverlay && selectedNodeId && !dismissed) {
      closeBtnRef.current?.focus();
    }
  }, [isOverlay, selectedNodeId, dismissed]);

  if (!selectedNodeId) return null;
  if (isOverlay && dismissed) return null;

  const node = nodes.find((n) => n.id === selectedNodeId);
  if (!node) return null;

  const step = stepResults.get(selectedNodeId);
  const showExecution =
    step && (executionStatus === 'running' || executionStatus === 'completed');

  const dialogProps = isOverlay
    ? ({
        role: 'dialog' as const,
        'aria-modal': true as const,
        'aria-label': 'Node inspector',
      })
    : {};

  return (
    <>
      {isOverlay && (
        <div
          className="clotho-backdrop"
          onClick={handleClose}
          aria-hidden="true"
        />
      )}
      <aside className="clotho-inspector" {...dialogProps}>
        <button
          ref={closeBtnRef}
          type="button"
          onClick={handleClose}
          className="clotho-inspector-close"
          aria-label="Close inspector"
        >
          {'\u2715'}
        </button>

        <h3
          style={{
            fontSize: 12,
            fontWeight: 600,
            textTransform: 'uppercase',
            color: 'var(--text-muted)',
            marginBottom: 14,
            letterSpacing: '0.04em',
          }}
        >
          Inspector
        </h3>

        {/* Execution overlay when running/completed */}
        {showExecution && step && (
          <div style={{ marginBottom: 16 }}>
            <ExecutionInspector step={step} />
            <hr
              style={{
                border: 'none',
                borderTop: '1px solid var(--surface-border)',
                marginTop: 14,
              }}
            />
          </div>
        )}

        {/* Node configuration */}
        {node.data.nodeType === 'agent' ? (
          <AgentInspector
            nodeId={node.id}
            label={node.data.label}
            config={node.data.config as AgentNodeConfig}
            stepResult={step}
          />
        ) : node.data.nodeType === 'media' ? (
          <MediaInspector
            nodeId={node.id}
            label={node.data.label}
            config={node.data.config as MediaNodeConfig}
            stepResult={step}
          />
        ) : (
          <ToolInspector
            nodeId={node.id}
            label={node.data.label}
            config={node.data.config as ToolNodeConfig}
          />
        )}

        {/* Delete node */}
        <div style={{ marginTop: 24, paddingTop: 14, borderTop: '1px solid #1e2030' }}>
          <button
            onClick={handleDelete}
            style={{
              width: '100%',
              padding: '8px 0',
              borderRadius: 6,
              border: '1px solid rgba(248, 113, 113, 0.3)',
              background: 'rgba(248, 113, 113, 0.08)',
              color: 'var(--status-failed)',
              fontSize: 12,
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            Delete Node
          </button>
        </div>
      </aside>
    </>
  );
}
