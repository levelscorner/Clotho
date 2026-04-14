import React, { useCallback } from 'react';
import { Handle, Position } from '@xyflow/react';
import type { Port, ExecutionStatus } from '../../../lib/types';
import { PORT_TYPE_LABEL } from '../../../lib/portCompatibility';
import { useExecutionStore } from '../../../stores/executionStore';
import { usePipelineStore } from '../../../stores/pipelineStore';

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
   * Human-readable node label, used for the delete button's aria-label.
   * Defaults to the node id if not provided, so legacy callers continue
   * to work without a breaking change.
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

  const statusClass = status ? ` clotho-node--${status}` : '';
  const selectedClass = selected ? ' selected' : '';
  const extraClass = className ? ` ${className}` : '';

  const inputPorts = ports.filter((p) => p.direction === 'input');
  const outputPorts = ports.filter((p) => p.direction === 'output');

  const handleDelete = useCallback(
    (e: React.MouseEvent<HTMLButtonElement>) => {
      e.stopPropagation();
      e.preventDefault();
      usePipelineStore.getState().removeNodes([id]);
    },
    [id],
  );

  const deleteLabel = `Delete node ${label ?? id}`;

  return (
    <div
      className={`clotho-node clotho-node--${variant}${statusClass}${selectedClass}${extraClass}`}
    >
      <button
        type="button"
        className="clotho-node__delete-btn"
        aria-label={deleteLabel}
        onClick={handleDelete}
        onMouseDown={(e) => e.stopPropagation()}
      >
        <svg
          className="clotho-node__delete-icon"
          viewBox="0 0 12 12"
          aria-hidden="true"
          focusable="false"
        >
          <path
            d="M3 3 L9 9 M9 3 L3 9"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
          />
        </svg>
      </button>

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
              title={`${port.name} (${port.type})`}
            />
            <span
              className="clotho-port-label clotho-port-label--in"
              style={{ top }}
              aria-hidden="true"
            >
              {port.name} · {typeLabel}
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
              title={`${port.name} (${port.type})`}
            />
            <span
              className="clotho-port-label clotho-port-label--out"
              style={{ top }}
              aria-hidden="true"
            >
              {port.name} · {typeLabel}
            </span>
          </React.Fragment>
        );
      })}
    </div>
  );
}

export const BaseNode = React.memo(BaseNodeInner);
