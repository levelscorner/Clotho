import React, { useMemo } from 'react';
import { Handle, Position, NodeResizer } from '@xyflow/react';
import { Lock as LockIcon } from 'phosphor-react';
import type { Port, ExecutionStatus } from '../../../lib/types';
import { PORT_TYPE_LABEL } from '../../../lib/portCompatibility';
import { useExecutionStore } from '../../../stores/executionStore';
import { usePipelineStore } from '../../../stores/pipelineStore';
import { NodeActionsMenu } from './NodeActionsMenu';
import { describeNode } from '../../../lib/nodeDescriptions';

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

  // Subscribe narrowly to this node's data so we can compute a teaser
  // description without reaching into the subclass components.
  const node = usePipelineStore((s) => s.nodes.find((n) => n.id === id));
  const data = node?.data as Record<string, unknown> | undefined;

  const teaser = useMemo(() => {
    if (!data) return null;
    const nt = data.nodeType as 'agent' | 'media' | 'tool' | undefined;
    if (nt !== 'agent' && nt !== 'media' && nt !== 'tool') return null;
    const cfg = (data.config ?? {}) as Record<string, unknown>;
    return describeNode({
      nodeType: nt,
      mediaType: cfg.media_type as string | undefined,
      toolType: cfg.tool_type as string | undefined,
      presetCategory: cfg.preset_category as string | undefined,
      presetDescription: data.description as string | undefined,
    }).teaser;
  }, [data]);

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
      {/* Resize handles on corners + edges; only visible when node is selected.
          React Flow tracks the new dimensions in its internal store and
          triggers edge reflow via ResizeObserver automatically. */}
      <NodeResizer
        color="var(--accent)"
        isVisible={Boolean(selected)}
        minWidth={220}
        minHeight={140}
        keepAspectRatio={false}
      />

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

      {teaser && (
        <p className="clotho-node__description" title={teaser}>
          {teaser}
        </p>
      )}

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
