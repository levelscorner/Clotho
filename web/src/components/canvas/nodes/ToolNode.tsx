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

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

type ToolNodeType = Node<ToolNodeData>;

function ToolNodeInner({ id, data, selected }: NodeProps<ToolNodeType>) {
  const setSelectedNode = usePipelineStore((s) => s.setSelectedNode);
  const updateNodeConfig = usePipelineStore((s) => s.updateNodeConfig);

  const handleClick = useCallback(() => {
    setSelectedNode(id);
  }, [id, setSelectedNode]);

  const handleContentChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      const val = e.target.value;
      updateNodeConfig(id, (prev) => ({ ...(prev as ToolNodeConfig), content: val }));
    },
    [id, updateNodeConfig],
  );

  const handleUrlChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const val = e.target.value;
      updateNodeConfig(id, (prev) => ({
        ...(prev as ToolNodeConfig),
        media_url: val,
      }));
    },
    [id, updateNodeConfig],
  );

  const config = data.config as ToolNodeConfig;
  const icon = TOOL_ICONS[config.tool_type] ?? '\u{1f527}';

  return (
    <div onClick={handleClick} role="button" tabIndex={0}>
      <BaseNode id={id} ports={data.ports} variant="tool" selected={selected}>
        <div className="clotho-node__header">
          <span style={{ fontSize: 16 }} aria-hidden>
            {icon}
          </span>
          <span className="clotho-node__label">{data.label}</span>
        </div>
        <div className="clotho-node__body">
          {config.tool_type === 'text_box' ? (
            <textarea
              className="clotho-node__prompt nodrag nowheel"
              value={config.content ?? ''}
              onChange={handleContentChange}
              onMouseDown={(e) => e.stopPropagation()}
              onKeyDown={(e) => e.stopPropagation()}
              placeholder="Type anything…"
              aria-label="Text content"
            />
          ) : (
            <input
              className="clotho-node__prompt clotho-node__prompt--inline nodrag nowheel"
              type="text"
              value={config.media_url ?? ''}
              onChange={handleUrlChange}
              onMouseDown={(e) => e.stopPropagation()}
              onKeyDown={(e) => e.stopPropagation()}
              placeholder="https://…"
              aria-label={`${config.tool_type} URL`}
            />
          )}
        </div>
      </BaseNode>
    </div>
  );
}

export const ToolNode = React.memo(ToolNodeInner);
