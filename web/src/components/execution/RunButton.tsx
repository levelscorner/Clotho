import { useCallback } from 'react';
import { usePipelineStore } from '../../stores/pipelineStore';
import { useExecutionStore } from '../../stores/executionStore';

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function RunButton() {
  const pipelineId = usePipelineStore((s) => s.pipelineId);
  const isDirty = usePipelineStore((s) => s.isDirty);
  const save = usePipelineStore((s) => s.save);
  const executionStatus = useExecutionStore((s) => s.status);
  const startExecution = useExecutionStore((s) => s.startExecution);

  const isRunning = executionStatus === 'running' || executionStatus === 'pending';
  const disabled = !pipelineId || isRunning;

  const handleRun = useCallback(async () => {
    if (!pipelineId) return;

    // Auto-save before running if there are unsaved changes
    if (isDirty) {
      await save();
    }

    await startExecution(pipelineId);
  }, [pipelineId, isDirty, save, startExecution]);

  return (
    <button
      onClick={handleRun}
      disabled={disabled}
      style={{
        padding: '6px 16px',
        borderRadius: 6,
        border: 'none',
        background: isRunning ? '#334155' : '#22c55e',
        color: '#fff',
        fontSize: 13,
        fontWeight: 600,
        cursor: disabled ? 'not-allowed' : 'pointer',
        opacity: disabled ? 0.6 : 1,
        display: 'flex',
        alignItems: 'center',
        gap: 6,
        transition: 'background 0.15s',
      }}
    >
      {isRunning ? (
        <>
          <span
            style={{
              display: 'inline-block',
              width: 14,
              height: 14,
              border: '2px solid rgba(255,255,255,0.3)',
              borderTopColor: '#fff',
              borderRadius: '50%',
              animation: 'spin 0.8s linear infinite',
            }}
          />
          Running...
        </>
      ) : (
        <>
          <span aria-hidden>&#x25B6;</span> Run
        </>
      )}
    </button>
  );
}
