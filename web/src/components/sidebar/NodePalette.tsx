import { useEffect, useState, useCallback, type DragEvent } from 'react';
import type {
  AgentPreset,
  AgentNodeConfig,
  ToolNodeConfig,
  MediaNodeConfig,
  MediaType,
  Port,
  NodeType,
  ToolType,
} from '../../lib/types';
import { api } from '../../lib/api';

// ---------------------------------------------------------------------------
// DnD helper
// ---------------------------------------------------------------------------

interface DragPayload {
  nodeType: NodeType;
  config: AgentNodeConfig | ToolNodeConfig | MediaNodeConfig;
  ports: Port[];
  label?: string;
}

function setDragData(event: DragEvent, payload: DragPayload): void {
  event.dataTransfer.setData(
    'application/clotho-node',
    JSON.stringify(payload),
  );
  event.dataTransfer.effectAllowed = 'move';
}

// ---------------------------------------------------------------------------
// Default configs
// ---------------------------------------------------------------------------

function blankAgentConfig(): AgentNodeConfig {
  return {
    provider: 'openai',
    model: 'gpt-4o',
    role: { system_prompt: '', persona: '' },
    task: { task_type: 'custom', output_type: 'text', template: '' },
    temperature: 0.7,
    max_tokens: 2048,
  };
}

function defaultAgentPorts(): Port[] {
  return [
    { id: 'in_text', name: 'Input', type: 'any', direction: 'input', required: false },
    { id: 'out_text', name: 'Output', type: 'text', direction: 'output', required: false },
  ];
}

function toolConfig(toolType: ToolType): ToolNodeConfig {
  return { tool_type: toolType, content: '' };
}

function toolPorts(toolType: ToolType): Port[] {
  const outputType =
    toolType === 'text_box' ? 'text' as const
      : toolType === 'image_box' ? 'image' as const
      : 'video' as const;

  return [
    {
      id: `out_${outputType}`,
      name: 'Output',
      type: outputType,
      direction: 'output',
      required: false,
    },
  ];
}

// ---------------------------------------------------------------------------
// Media palette items
// ---------------------------------------------------------------------------

interface MediaItem {
  label: string;
  initials: string;
  mediaType: MediaType;
  defaultConfig: MediaNodeConfig;
  ports: Port[];
}

const MEDIA_ITEMS: MediaItem[] = [
  {
    label: 'Image Gen',
    initials: 'Ig',
    mediaType: 'image',
    defaultConfig: {
      media_type: 'image',
      provider: 'replicate',
      model: 'flux-1.1-pro',
      prompt: '',
      aspect_ratio: '1:1',
      num_outputs: 1,
    },
    ports: [
      { id: 'in_image_prompt', name: 'Prompt', type: 'image_prompt', direction: 'input', required: true },
      { id: 'out_image', name: 'Image', type: 'image', direction: 'output', required: false },
    ],
  },
  {
    label: 'Video Gen',
    initials: 'Vg',
    mediaType: 'video',
    defaultConfig: {
      media_type: 'video',
      provider: 'replicate',
      model: 'stable-video-diffusion',
      prompt: '',
      aspect_ratio: '16:9',
    },
    ports: [
      { id: 'in_video_prompt', name: 'Prompt', type: 'video_prompt', direction: 'input', required: true },
      { id: 'out_video', name: 'Video', type: 'video', direction: 'output', required: false },
    ],
  },
  {
    label: 'Voice TTS',
    initials: 'Tt',
    mediaType: 'audio',
    defaultConfig: {
      media_type: 'audio',
      provider: 'openai',
      model: 'tts-1',
      prompt: '',
      voice: 'alloy',
    },
    ports: [
      { id: 'in_audio_prompt', name: 'Prompt', type: 'audio_prompt', direction: 'input', required: true },
      { id: 'out_audio', name: 'Audio', type: 'audio', direction: 'output', required: false },
    ],
  },
];

// ---------------------------------------------------------------------------
// Tool items
// ---------------------------------------------------------------------------

interface ToolItem {
  label: string;
  initials: string;
  toolType: ToolType;
}

const TOOLS: ToolItem[] = [
  { label: 'Text', initials: 'Tx', toolType: 'text_box' },
  { label: 'Image', initials: 'Im', toolType: 'image_box' },
  { label: 'Video', initials: 'Vi', toolType: 'video_box' },
];

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

const sectionLabel: React.CSSProperties = {
  fontSize: 10,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.8px',
  color: 'var(--text-muted)',
  padding: '12px 0 6px',
};

const gridStyle: React.CSSProperties = {
  display: 'grid',
  gridTemplateColumns: 'repeat(3, 1fr)',
  gap: 6,
};

const tileStyle: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  gap: 4,
  padding: '10px 4px 8px',
  borderRadius: 'var(--radius-sm)',
  cursor: 'grab',
  userSelect: 'none',
  border: '1px solid transparent',
  background: 'var(--surface-raised)',
  transition: 'all var(--duration-fast)',
};

const tileIconStyle = (bg: string, color: string): React.CSSProperties => ({
  width: 32,
  height: 32,
  borderRadius: 'var(--radius-sm)',
  background: bg,
  color: color,
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  fontSize: 12,
  fontWeight: 700,
  fontFamily: 'var(--font-mono)',
});

const tileLabelStyle: React.CSSProperties = {
  fontSize: 9,
  color: 'var(--text-secondary)',
  textAlign: 'center',
  lineHeight: 1.3,
  maxWidth: '100%',
  wordBreak: 'break-word',
  hyphens: 'auto',
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function NodePalette() {
  const [presets, setPresets] = useState<AgentPreset[]>([]);

  useEffect(() => {
    api
      .get<AgentPreset[]>('/presets')
      .then((data) => setPresets(Array.isArray(data) ? data : []))
      .catch(() => {});
  }, []);

  const onDragStart = useCallback(
    (event: DragEvent, payload: DragPayload) => {
      setDragData(event, payload);
    },
    [],
  );

  const handleHover = (e: React.MouseEvent, hovering: boolean) => {
    const el = e.currentTarget as HTMLDivElement;
    el.style.borderColor = hovering ? 'var(--surface-border)' : 'transparent';
    el.style.background = hovering ? 'var(--surface-hover)' : 'var(--surface-raised)';
  };

  // Generate unique initials from preset name
  const getInitials = useCallback((name: string, allNames: string[]) => {
    const words = name.split(/\s+/).filter(Boolean);
    if (words.length < 2) return name.slice(0, 2);

    const simple = `${words[0][0].toUpperCase()}${words[1][0].toLowerCase()}`;

    // Check for collisions with other presets
    const hasCollision = allNames.some((other) => {
      if (other === name) return false;
      const ow = other.split(/\s+/).filter(Boolean);
      if (ow.length < 2) return false;
      return `${ow[0][0].toUpperCase()}${ow[1][0].toLowerCase()}` === simple;
    });

    if (!hasCollision) return simple;

    // Use first two chars of the first word to disambiguate
    return words[0].slice(0, 2).charAt(0).toUpperCase() + words[0].slice(1, 2).toLowerCase();
  }, []);

  return (
    <aside
      aria-label="Node palette"
      style={{
        width: 210,
        minWidth: 210,
        height: '100%',
        background: 'var(--surface-base)',
        borderRight: '1px solid var(--surface-border)',
        overflowY: 'auto',
        padding: '8px 10px',
      }}
    >
      {/* ---- AGENT ---- */}
      <h3 style={sectionLabel}>Agent</h3>
      <div style={gridStyle}>
        {/* Primary: blank agent */}
        <div
          draggable
          onDragStart={(e) =>
            onDragStart(e, {
              nodeType: 'agent',
              config: blankAgentConfig(),
              ports: defaultAgentPorts(),
              label: 'Agent',
            })
          }
          style={{ ...tileStyle, gridColumn: 'span 3', flexDirection: 'row', gap: 8, padding: '8px 10px' }}
          onMouseOver={(e) => handleHover(e, true)}
          onMouseOut={(e) => handleHover(e, false)}
        >
          <div style={tileIconStyle('var(--accent-soft)', 'var(--accent)')}>+</div>
          <div>
            <div style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-primary)' }}>New Agent</div>
            <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>Drag to canvas</div>
          </div>
        </div>
      </div>

      {/* ---- PERSONALITIES ---- */}
      {presets.length > 0 && (
        <>
          <h3 style={{ ...sectionLabel, fontSize: 9 }}>Personalities</h3>
          <div style={gridStyle}>
            {presets.map((preset) => (
              <div
                key={preset.id}
                draggable
                onDragStart={(e) =>
                  onDragStart(e, {
                    nodeType: 'agent',
                    config: preset.config,
                    ports: defaultAgentPorts(),
                    label: preset.name,
                  })
                }
                style={tileStyle}
                onMouseOver={(e) => handleHover(e, true)}
                onMouseOut={(e) => handleHover(e, false)}
                title={preset.name}
              >
                <div style={tileIconStyle('var(--accent-soft)', 'var(--accent)')}>
                  {getInitials(preset.name, presets.map((p) => p.name))}
                </div>
                <div style={tileLabelStyle}>{preset.name}</div>
              </div>
            ))}
          </div>
        </>
      )}

      {/* ---- MEDIA ---- */}
      <h3 style={sectionLabel}>Media</h3>
      <div style={gridStyle}>
        {MEDIA_ITEMS.map((item) => {
          const colorMap: Record<MediaType, { bg: string; fg: string }> = {
            image: { bg: 'rgba(245, 158, 11, 0.15)', fg: 'var(--port-image)' },
            video: { bg: 'rgba(239, 68, 68, 0.15)', fg: 'var(--port-video)' },
            audio: { bg: 'rgba(6, 182, 212, 0.15)', fg: 'var(--port-audio)' },
          };
          const c = colorMap[item.mediaType];
          return (
            <div
              key={item.mediaType}
              draggable
              onDragStart={(e) =>
                onDragStart(e, {
                  nodeType: 'media',
                  config: item.defaultConfig,
                  ports: item.ports,
                  label: item.label,
                })
              }
              style={tileStyle}
              onMouseOver={(e) => handleHover(e, true)}
              onMouseOut={(e) => handleHover(e, false)}
              title={item.label}
            >
              <div style={tileIconStyle(c.bg, c.fg)}>{item.initials}</div>
              <div style={tileLabelStyle}>{item.label}</div>
            </div>
          );
        })}
      </div>

      {/* ---- TOOLS ---- */}
      <h3 style={sectionLabel}>Tools</h3>
      <div style={gridStyle}>
        {TOOLS.map((item) => (
          <div
            key={item.toolType}
            draggable
            onDragStart={(e) =>
              onDragStart(e, {
                nodeType: 'tool',
                config: toolConfig(item.toolType),
                ports: toolPorts(item.toolType),
                label: item.label,
              })
            }
            style={tileStyle}
            onMouseOver={(e) => handleHover(e, true)}
            onMouseOut={(e) => handleHover(e, false)}
            title={item.label}
          >
            <div style={tileIconStyle('rgba(136, 136, 160, 0.15)', 'var(--text-secondary)')}>
              {item.initials}
            </div>
            <div style={tileLabelStyle}>{item.label}</div>
          </div>
        ))}
      </div>
    </aside>
  );
}
