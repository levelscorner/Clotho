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
// Tool palette items
// ---------------------------------------------------------------------------

interface ToolItem {
  label: string;
  toolType: ToolType;
  icon: string;
}

const TOOLS: ToolItem[] = [
  { label: 'Text Box', toolType: 'text_box', icon: '\u{1f4dd}' },
  { label: 'Image Box', toolType: 'image_box', icon: '\u{1f5bc}' },
  { label: 'Video Box', toolType: 'video_box', icon: '\u{1f3ac}' },
];

// ---------------------------------------------------------------------------
// Media palette items
// ---------------------------------------------------------------------------

interface MediaItem {
  label: string;
  mediaType: MediaType;
  icon: string;
  defaultConfig: MediaNodeConfig;
  ports: Port[];
}

const MEDIA_ITEMS: MediaItem[] = [
  {
    label: 'Image Generator',
    mediaType: 'image',
    icon: '\u{1F4F7}',
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
    label: 'Video Generator',
    mediaType: 'video',
    icon: '\u{1F3AC}',
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
    label: 'Voice / TTS',
    mediaType: 'audio',
    icon: '\u{1F50A}',
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
// Styles
// ---------------------------------------------------------------------------

const sectionTitle: React.CSSProperties = {
  fontSize: 11,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  color: '#64748b',
  padding: '12px 14px 6px',
};

const cardStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  padding: '8px 14px',
  margin: '2px 8px',
  borderRadius: 6,
  cursor: 'grab',
  userSelect: 'none',
  fontSize: 13,
  color: '#cbd5e1',
  background: '#1a1c2e',
  border: '1px solid transparent',
  transition: 'border-color 0.15s',
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function NodePalette() {
  const [presets, setPresets] = useState<AgentPreset[]>([]);

  useEffect(() => {
    api
      .get<AgentPreset[]>('/presets')
      .then(setPresets)
      .catch(() => {
        // API may not be available yet; continue with empty presets
      });
  }, []);

  const onPresetDragStart = useCallback(
    (event: DragEvent, preset: AgentPreset) => {
      setDragData(event, {
        nodeType: 'agent',
        config: preset.config,
        ports: defaultAgentPorts(),
        label: preset.name,
      });
    },
    [],
  );

  const onBlankAgentDragStart = useCallback((event: DragEvent) => {
    setDragData(event, {
      nodeType: 'agent',
      config: blankAgentConfig(),
      ports: defaultAgentPorts(),
      label: 'Blank Agent',
    });
  }, []);

  const onToolDragStart = useCallback((event: DragEvent, item: ToolItem) => {
    setDragData(event, {
      nodeType: 'tool',
      config: toolConfig(item.toolType),
      ports: toolPorts(item.toolType),
      label: item.label,
    });
  }, []);

  const onMediaDragStart = useCallback((event: DragEvent, item: MediaItem) => {
    setDragData(event, {
      nodeType: 'media',
      config: item.defaultConfig,
      ports: item.ports,
      label: item.label,
    });
  }, []);

  return (
    <aside
      style={{
        width: 250,
        minWidth: 250,
        height: '100%',
        background: '#12131f',
        borderRight: '1px solid #1e2030',
        overflowY: 'auto',
      }}
    >
      {/* Agents section */}
      <div style={sectionTitle}>Agents</div>

      <div
        draggable
        onDragStart={onBlankAgentDragStart}
        style={cardStyle}
        onMouseOver={(e) => {
          (e.currentTarget as HTMLDivElement).style.borderColor = '#334155';
        }}
        onMouseOut={(e) => {
          (e.currentTarget as HTMLDivElement).style.borderColor = 'transparent';
        }}
      >
        <span style={{ fontSize: 16 }} aria-hidden>
          &#x2795;
        </span>
        Blank Agent
      </div>

      {presets.map((preset) => (
        <div
          key={preset.id}
          draggable
          onDragStart={(e) => onPresetDragStart(e, preset)}
          style={cardStyle}
          onMouseOver={(e) => {
            (e.currentTarget as HTMLDivElement).style.borderColor = '#334155';
          }}
          onMouseOut={(e) => {
            (e.currentTarget as HTMLDivElement).style.borderColor =
              'transparent';
          }}
        >
          <span style={{ fontSize: 16 }} aria-hidden>
            {preset.icon || '\u{1f916}'}
          </span>
          {preset.name}
        </div>
      ))}

      {/* Tools section */}
      <div style={{ ...sectionTitle, marginTop: 8 }}>Tools</div>

      {TOOLS.map((item) => (
        <div
          key={item.toolType}
          draggable
          onDragStart={(e) => onToolDragStart(e, item)}
          style={cardStyle}
          onMouseOver={(e) => {
            (e.currentTarget as HTMLDivElement).style.borderColor = '#334155';
          }}
          onMouseOut={(e) => {
            (e.currentTarget as HTMLDivElement).style.borderColor =
              'transparent';
          }}
        >
          <span style={{ fontSize: 16 }} aria-hidden>
            {item.icon}
          </span>
          {item.label}
        </div>
      ))}

      {/* Media section */}
      <div style={{ ...sectionTitle, marginTop: 8 }}>Media</div>

      {MEDIA_ITEMS.map((item) => (
        <div
          key={item.mediaType}
          draggable
          onDragStart={(e) => onMediaDragStart(e, item)}
          style={cardStyle}
          onMouseOver={(e) => {
            (e.currentTarget as HTMLDivElement).style.borderColor = '#334155';
          }}
          onMouseOut={(e) => {
            (e.currentTarget as HTMLDivElement).style.borderColor =
              'transparent';
          }}
        >
          <span style={{ fontSize: 16 }} aria-hidden>
            {item.icon}
          </span>
          {item.label}
        </div>
      ))}
    </aside>
  );
}
