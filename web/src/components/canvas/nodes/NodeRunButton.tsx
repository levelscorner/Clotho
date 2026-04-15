import React, { useCallback } from 'react';
import { Play, CircleNotch } from 'phosphor-react';
import { usePipelineStore } from '../../../stores/pipelineStore';
import { useExecutionStore } from '../../../stores/executionStore';

// ---------------------------------------------------------------------------
// NodeRunButton
//
// Small play button shown in the footer of executable nodes (Agent, Media).
// Click starts the pipeline from this node — if no execution exists yet we
// start a fresh run; if one exists we retry just this step. While the node
// is running, the button is replaced by a spinner-only indicator (no button)
// per the design intent "progress indicator only when actively processing."
// ---------------------------------------------------------------------------

interface NodeRunButtonProps {
  nodeId: string;
}

function NodeRunButtonInner({ nodeId }: NodeRunButtonProps) {
  const pipelineId = usePipelineStore((s) => s.pipelineId);
  const isDirty = usePipelineStore((s) => s.isDirty);
  const save = usePipelineStore((s) => s.save);
  const isLocked = usePipelineStore((s) => s.lockedNodes.has(nodeId));
  const stepResult = useExecutionStore((s) => s.stepResults.get(nodeId));
  const executionId = useExecutionStore((s) => s.executionId);
  const startExecution = useExecutionStore((s) => s.startExecution);
  const retryNode = useExecutionStore((s) => s.retryNode);

  const status = stepResult?.status;
  const isRunning = status === 'running' || status === 'pending';

  const handleRun = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      if (!pipelineId || isLocked) return;
      if (isDirty) await save();
      if (executionId) {
        retryNode(executionId, nodeId);
      } else {
        await startExecution(pipelineId, nodeId);
      }
    },
    [
      pipelineId,
      isLocked,
      isDirty,
      save,
      executionId,
      retryNode,
      startExecution,
      nodeId,
    ],
  );

  if (isRunning) {
    return (
      <span
        className="clotho-node__run clotho-node__run--progress"
        role="status"
        aria-label="Running"
        aria-live="polite"
      >
        <CircleNotch size={14} weight="bold" className="clotho-node__run-spin" />
      </span>
    );
  }

  return (
    <button
      type="button"
      className="clotho-node__run"
      onClick={handleRun}
      onMouseDown={(e) => e.stopPropagation()}
      disabled={!pipelineId || isLocked}
      aria-label="Run this step"
      title={executionId ? 'Re-run this step' : 'Run from this step'}
    >
      <Play size={12} weight="fill" aria-hidden="true" />
    </button>
  );
}

export const NodeRunButton = React.memo(NodeRunButtonInner);
