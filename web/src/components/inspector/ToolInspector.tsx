import { useCallback } from 'react';
import type { ToolNodeConfig } from '../../lib/types';
import { usePipelineStore } from '../../stores/pipelineStore';
import { InspectorGroup } from './InspectorGroup';
import { AboutNodeSection } from './AboutNodeSection';
import { describeNode } from '../../lib/nodeDescriptions';

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

const fieldGroup: React.CSSProperties = {
  marginBottom: 12,
};

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: 11,
  fontWeight: 600,
  color: '#64748b',
  marginBottom: 4,
  textTransform: 'uppercase',
  letterSpacing: '0.04em',
};

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '6px 8px',
  borderRadius: 4,
  border: '1px solid #334155',
  background: '#1a1c2e',
  color: '#e2e8f0',
  fontSize: 13,
};

const textareaStyle: React.CSSProperties = {
  ...inputStyle,
  minHeight: 120,
  resize: 'vertical',
};

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface ToolInspectorProps {
  nodeId: string;
  label: string;
  config: ToolNodeConfig;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function ToolInspector({ nodeId, label, config }: ToolInspectorProps) {
  const updateNodeConfig = usePipelineStore((s) => s.updateNodeConfig);
  const updateNodeLabel = usePipelineStore((s) => s.updateNodeLabel);

  const update = useCallback(
    (patch: Partial<ToolNodeConfig>) => {
      updateNodeConfig(nodeId, (prev) => ({ ...prev, ...patch }));
    },
    [nodeId, updateNodeConfig],
  );

  return (
    <div>
      <AboutNodeSection
        description={describeNode({
          nodeType: 'tool',
          toolType: (config as { tool_type?: string }).tool_type,
        })}
      />

      <InspectorGroup title="Basics" defaultOpen>
        <div style={fieldGroup}>
          <label style={labelStyle}>Label</label>
          <input
            style={inputStyle}
            value={label}
            onChange={(e) => updateNodeLabel(nodeId, e.target.value)}
          />
        </div>

        {config.tool_type === 'text_box' && (
          <div style={fieldGroup}>
            <label style={labelStyle}>Content</label>
            <textarea
              style={textareaStyle}
              value={config.content ?? ''}
              onChange={(e) => update({ content: e.target.value })}
              placeholder="Enter text content..."
            />
          </div>
        )}

        {(config.tool_type === 'image_box' ||
          config.tool_type === 'video_box') && (
          <div style={fieldGroup}>
            <label style={labelStyle}>Media URL</label>
            <input
              style={inputStyle}
              value={config.media_url ?? ''}
              onChange={(e) => update({ media_url: e.target.value })}
              placeholder="https://..."
            />
          </div>
        )}
      </InspectorGroup>

      <InspectorGroup title="Advanced">
        <div
          style={{
            fontSize: 12,
            color: 'var(--text-muted)',
            padding: '4px 0',
          }}
        >
          No advanced settings for this tool.
        </div>
      </InspectorGroup>
    </div>
  );
}
