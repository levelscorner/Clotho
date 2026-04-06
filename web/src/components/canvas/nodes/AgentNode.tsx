import React, { useCallback } from 'react';
import type { NodeProps, Node } from '@xyflow/react';
import type { AgentNodeData, AgentNodeConfig } from '../../../lib/types';
import { BaseNode } from './BaseNode';
import { usePipelineStore } from '../../../stores/pipelineStore';

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

type AgentNodeType = Node<AgentNodeData>;

function AgentNodeInner({ id, data, selected }: NodeProps<AgentNodeType>) {
  const setSelectedNode = usePipelineStore((s) => s.setSelectedNode);

  const handleClick = useCallback(() => {
    setSelectedNode(id);
  }, [id, setSelectedNode]);

  const config = data.config as AgentNodeConfig;

  return (
    <div onClick={handleClick} role="button" tabIndex={0} onKeyDown={handleClick}>
      <BaseNode
        id={id}
        ports={data.ports}
        variant="agent"
        selected={selected}
      >
        <div className="clotho-node__header">
          <span style={{ fontSize: 16 }} aria-hidden>
            &#x1f916;
          </span>
          <span className="clotho-node__label">
            {config.role.persona || data.label}
          </span>
        </div>
        <div className="clotho-node__body">
          <span className="clotho-node__badge">{config.model}</span>{' '}
          <span className="clotho-node__badge">{config.task.task_type}</span>
        </div>
      </BaseNode>
    </div>
  );
}

export const AgentNode = React.memo(AgentNodeInner);
