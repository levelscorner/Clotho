import React, { useCallback } from 'react';
import type { NodeProps, Node } from '@xyflow/react';
import type { AgentNodeData, AgentNodeConfig } from '../../../lib/types';
import { BaseNode } from './BaseNode';
import { usePipelineStore } from '../../../stores/pipelineStore';
import { useExecutionStore } from '../../../stores/executionStore';

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

type AgentNodeType = Node<AgentNodeData>;

function AgentNodeInner({ id, data, selected }: NodeProps<AgentNodeType>) {
  const setSelectedNode = usePipelineStore((s) => s.setSelectedNode);
  const stepResult = useExecutionStore((s) => s.stepResults.get(id));
  const executionId = useExecutionStore((s) => s.executionId);
  const retryNode = useExecutionStore((s) => s.retryNode);

  const handleClick = useCallback(() => {
    setSelectedNode(id);
  }, [id, setSelectedNode]);

  const handleRetry = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      if (executionId) retryNode(executionId, id);
    },
    [executionId, id, retryNode],
  );

  const handleEditPrompt = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      setSelectedNode(id);
    },
    [id, setSelectedNode],
  );

  const config = data.config as AgentNodeConfig;
  const status = stepResult?.status;
  const output = stepResult?.output;
  const error = stepResult?.error;
  const cost = stepResult?.cost;
  const duration = stepResult?.duration_ms;

  // Generate 2-letter initials from label (e.g., "Script Writer" -> "Sw")
  const label = config.role.persona || data.label;
  const words = label.split(/\s+/).filter(Boolean);
  const initials =
    words.length >= 2
      ? `${words[0][0].toUpperCase()}${words[1][0].toLowerCase()}`
      : label.slice(0, 2);

  return (
    <div
      onClick={handleClick}
      role="button"
      tabIndex={0}
      aria-label={`${label} agent, status: ${status || 'idle'}`}
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleClick(); } }}
    >
      <BaseNode
        id={id}
        ports={data.ports}
        variant="agent"
        selected={selected}
      >
        <div className="clotho-node__header">
          <div className="clotho-node__icon clotho-node__icon--agent">
            {initials}
          </div>
          <div>
            <div className="clotho-node__title">{label}</div>
            <div className="clotho-node__subtitle">{config.task.task_type}</div>
          </div>
        </div>

        <div className="clotho-node__body">
          {/* Streaming output preview */}
          {output && status === 'running' ? (
            <div className="clotho-node__preview clotho-node__preview--streaming">
              {output.slice(-80)}
              <span className="clotho-node__cursor" />
            </div>
          ) : output && status === 'completed' ? (
            <div className="clotho-node__preview">
              {output.slice(0, 100)}
              {output.length > 100 ? '...' : ''}
            </div>
          ) : status === 'failed' ? (
            <>
              <div className="clotho-node__error">
                {error || 'Execution failed'}
              </div>
              <div className="clotho-node__error-actions">
                <button
                  className="clotho-node__error-btn clotho-node__error-btn--primary"
                  onClick={handleRetry}
                >
                  Retry
                </button>
                <button
                  className="clotho-node__error-btn"
                  onClick={handleEditPrompt}
                >
                  Edit Prompt
                </button>
              </div>
            </>
          ) : (
            <div className="clotho-node__preview" style={{ fontStyle: 'italic' }}>
              Ready to execute
            </div>
          )}
        </div>

        {/* Footer with status + cost */}
        <div className="clotho-node__footer">
          <span className={`clotho-node__status-dot clotho-node__status-dot--${status ?? 'idle'}`} />
          <span>{status ?? 'Idle'}</span>
          {duration != null && <span>&middot; {(duration / 1000).toFixed(1)}s</span>}
          {cost != null && <span>&middot; ${cost.toFixed(4)}</span>}
        </div>
      </BaseNode>
    </div>
  );
}

export const AgentNode = React.memo(AgentNodeInner);
