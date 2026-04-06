import React, { useCallback } from 'react';
import type { NodeProps, Node } from '@xyflow/react';
import type { ToolNodeData, ToolNodeConfig } from '../../../lib/types';
import { BaseNode } from './BaseNode';
import { usePipelineStore } from '../../../stores/pipelineStore';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const TOOL_ICONS: Record<string, string> = {
  text_box: '\u{1f4dd}',
  image_box: '\u{1f5bc}',
  video_box: '\u{1f3ac}',
};

function truncate(text: string, max: number): string {
  return text.length > max ? text.slice(0, max) + '\u2026' : text;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

type ToolNodeType = Node<ToolNodeData>;

function ToolNodeInner({ id, data, selected }: NodeProps<ToolNodeType>) {
  const setSelectedNode = usePipelineStore((s) => s.setSelectedNode);

  const handleClick = useCallback(() => {
    setSelectedNode(id);
  }, [id, setSelectedNode]);

  const config = data.config as ToolNodeConfig;
  const icon = TOOL_ICONS[config.tool_type] ?? '\u{1f527}';

  return (
    <div onClick={handleClick} role="button" tabIndex={0} onKeyDown={handleClick}>
      <BaseNode id={id} ports={data.ports} variant="tool" selected={selected}>
        <div className="clotho-node__header">
          <span style={{ fontSize: 16 }} aria-hidden>
            {icon}
          </span>
          <span className="clotho-node__label">{data.label}</span>
        </div>
        <div className="clotho-node__body">
          {config.content ? (
            <span
              style={{ fontSize: 11, color: '#94a3b8', whiteSpace: 'pre-wrap' }}
            >
              {truncate(config.content, 80)}
            </span>
          ) : config.media_url ? (
            <span className="clotho-node__badge">media</span>
          ) : null}
        </div>
      </BaseNode>
    </div>
  );
}

export const ToolNode = React.memo(ToolNodeInner);
