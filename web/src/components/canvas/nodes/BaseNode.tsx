import React from 'react';
import { Handle, Position } from '@xyflow/react';
import type { Port, ExecutionStatus } from '../../../lib/types';
import { PORT_COLORS } from '../../../lib/portCompatibility';
import { useExecutionStore } from '../../../stores/executionStore';

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface BaseNodeProps {
  id: string;
  children: React.ReactNode;
  ports: Port[];
  variant: 'agent' | 'tool';
  selected?: boolean;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

function BaseNodeInner({
  id,
  children,
  ports,
  variant,
  selected,
}: BaseNodeProps) {
  const stepResult = useExecutionStore((s) => s.stepResults.get(id));
  const status: ExecutionStatus | undefined = stepResult?.status;

  const statusClass = status ? ` clotho-node--${status}` : '';
  const selectedClass = selected ? ' selected' : '';

  const inputPorts = ports.filter((p) => p.direction === 'input');
  const outputPorts = ports.filter((p) => p.direction === 'output');

  return (
    <div
      className={`clotho-node clotho-node--${variant}${statusClass}${selectedClass}`}
    >
      {/* Input handles */}
      {inputPorts.map((port, i) => (
        <Handle
          key={port.id}
          id={port.id}
          type="target"
          position={Position.Left}
          style={{
            background: PORT_COLORS[port.type],
            top: `${((i + 1) / (inputPorts.length + 1)) * 100}%`,
          }}
          title={`${port.name} (${port.type})`}
        />
      ))}

      {children}

      {/* Output handles */}
      {outputPorts.map((port, i) => (
        <Handle
          key={port.id}
          id={port.id}
          type="source"
          position={Position.Right}
          style={{
            background: PORT_COLORS[port.type],
            top: `${((i + 1) / (outputPorts.length + 1)) * 100}%`,
          }}
          title={`${port.name} (${port.type})`}
        />
      ))}
    </div>
  );
}

export const BaseNode = React.memo(BaseNodeInner);
