import { useCallback } from 'react';
import { usePipelineStore } from '../../stores/pipelineStore';
import { useExecutionStore } from '../../stores/executionStore';
import { api } from '../../lib/api';

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function RunButton() {
  const pipelineId = usePipelineStore((s) => s.pipelineId);
  const isDirty = usePipelineStore((s) => s.isDirty);
  const save = usePipelineStore((s) => s.save);
  const executionStatus = useExecutionStore((s) => s.status);
  const executionId = useExecutionStore((s) => s.executionId);
  const startExecution = useExecutionStore((s) => s.startExecution);
  const reset = useExecutionStore((s) => s.reset);

  const isRunning = executionStatus === 'running' || executionStatus === 'pending';

  const handleRun = useCallback(async () => {
    if (!pipelineId) return;
    if (isDirty) await save();
    await startExecution(pipelineId);
  }, [pipelineId, isDirty, save, startExecution]);

  const handleCancel = useCallback(async () => {
    if (!executionId) return;
    try {
      await api.post(`/executions/${executionId}/cancel`);
    } catch {
      // best-effort cancel
    }
    reset();
  }, [executionId, reset]);

  if (isRunning) {
    return (
      <button
        onClick={handleCancel}
        style={{
          padding: '6px 16px',
          borderRadius: 6,
          border: '1px solid #f87171',
          background: 'rgba(248, 113, 113, 0.12)',
          color: '#f87171',
          fontSize: 13,
          fontWeight: 600,
          cursor: 'pointer',
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          transition: 'background 0.15s',
        }}
      >
        <span aria-hidden>&#x25A0;</span> Stop
      </button>
    );
  }

  return (
    <button
      onClick={handleRun}
      disabled={!pipelineId}
      style={{
        padding: '6px 16px',
        borderRadius: 6,
        border: 'none',
        background: '#e5a84b',
        color: '#121216',
        fontSize: 13,
        fontWeight: 600,
        cursor: !pipelineId ? 'not-allowed' : 'pointer',
        opacity: !pipelineId ? 0.6 : 1,
        display: 'flex',
        alignItems: 'center',
        gap: 6,
        transition: 'background 0.15s',
      }}
    >
      <span aria-hidden>&#x25B6;</span> Run
    </button>
  );
}
