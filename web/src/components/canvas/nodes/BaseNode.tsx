import React from 'react';
import { Handle, Position } from '@xyflow/react';
import { Lock as LockIcon } from 'phosphor-react';
import type { Port, ExecutionStatus } from '../../../lib/types';
import { PORT_TYPE_LABEL } from '../../../lib/portCompatibility';
import { useExecutionStore } from '../../../stores/executionStore';
import { usePipelineStore } from '../../../stores/pipelineStore';
import { NodeActionsMenu } from './NodeActionsMenu';

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface BaseNodeProps {
  id: string;
  children: React.ReactNode;
  ports: Port[];
  variant: 'agent' | 'tool' | 'media';
  selected?: boolean;
  className?: string;
  /**
   * Human-readable node label — forwarded to the actions menu for an
   * accessible trigger label. Defaults to the node id if not provided.
   */
  label?: string;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function portTopPercent(index: number, total: number): string {
  return `${((index + 1) / (total + 1)) * 100}%`;
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
  className,
  label,
}: BaseNodeProps) {
  const stepResult = useExecutionStore((s) => s.stepResults.get(id));
  const status: ExecutionStatus | undefined = stepResult?.status;
  const isLocked = usePipelineStore((s) => s.lockedNodes.has(id));

  const statusClass = status ? ` clotho-node--${status}` : '';
  const selectedClass = selected ? ' selected' : '';
  const lockedClass = isLocked ? ' clotho-node--locked' : '';
  const extraClass = className ? ` ${className}` : '';

  const inputPorts = ports.filter((p) => p.direction === 'input');
  const outputPorts = ports.filter((p) => p.direction === 'output');

  return (
    <div
      className={`clotho-node clotho-node--${variant}${statusClass}${selectedClass}${lockedClass}${extraClass}`}
    >
      {isLocked && (
        <span className="clotho-node__lock-badge" aria-label="Locked" title="Locked">
          <LockIcon size={14} weight="fill" />
        </span>
      )}

      <NodeActionsMenu nodeId={id} label={label} />

      {/* Input handles + hover labels */}
      {inputPorts.map((port, i) => {
        const top = portTopPercent(i, inputPorts.length);
        const typeLabel = PORT_TYPE_LABEL[port.type] ?? port.type;
        return (
          <React.Fragment key={port.id}>
            <Handle
              id={port.id}
              type="target"
              position={Position.Left}
              className={`clotho-handle clotho-handle--${port.type}`}
              style={{ top }}
              title={`${port.name} · ${typeLabel}`}
            />
            <span
              className="clotho-port-label clotho-port-label--in"
              style={{ top }}
              title={typeLabel}
              aria-hidden="true"
            >
              {port.name}{port.required ? '*' : ''}
            </span>
          </React.Fragment>
        );
      })}

      {children}

      {/* Output handles + hover labels */}
      {outputPorts.map((port, i) => {
        const top = portTopPercent(i, outputPorts.length);
        const typeLabel = PORT_TYPE_LABEL[port.type] ?? port.type;
        return (
          <React.Fragment key={port.id}>
            <Handle
              id={port.id}
              type="source"
              position={Position.Right}
              className={`clotho-handle clotho-handle--${port.type}`}
              style={{ top }}
              title={`${port.name} · ${typeLabel}`}
            />
            <span
              className="clotho-port-label clotho-port-label--out"
              style={{ top }}
              title={typeLabel}
              aria-hidden="true"
            >
              {port.name}{port.required ? '*' : ''}
            </span>
          </React.Fragment>
        );
      })}
    </div>
  );
}

export const BaseNode = React.memo(BaseNodeInner);
