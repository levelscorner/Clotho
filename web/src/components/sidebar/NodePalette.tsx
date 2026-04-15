import {
  useEffect,
  useRef,
  useState,
  useCallback,
  type DragEvent,
  type ComponentType,
} from 'react';
import {
  Robot,
  UserCircle,
  FrameCorners,
  Wrench,
  MagicWand,
  ImageSquare,
  VideoCamera,
  SpeakerHigh,
  TextAa,
  Image as ImageIcon,
  FilmStrip,
} from 'phosphor-react';
import type { IconProps, Icon } from 'phosphor-react';
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
import { useUIStore } from '../../stores/uiStore';
import { MobileHamburger, SmallScreenBanner, PhoneHint } from './MobileHamburger';
import { presetIcon } from '../../lib/presetIcons';

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
  icon: Icon;
  mediaType: MediaType;
  defaultConfig: MediaNodeConfig;
  ports: Port[];
}

const MEDIA_ITEMS: MediaItem[] = [
  {
    label: 'Image Gen',
    icon: ImageSquare,
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
    icon: VideoCamera,
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
    icon: SpeakerHigh,
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
  icon: Icon;
  toolType: ToolType;
}

const TOOLS: ToolItem[] = [
  { label: 'Text', icon: TextAa, toolType: 'text_box' },
  { label: 'Image', icon: ImageIcon, toolType: 'image_box' },
  { label: 'Video', icon: FilmStrip, toolType: 'video_box' },
];

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

const sectionLabelStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 6,
  fontSize: 10,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.8px',
  color: 'var(--text-muted)',
  padding: '12px 0 6px',
  margin: 0,
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
  justifyContent: 'center',
  gap: 6,
  padding: '10px 4px 8px',
  borderRadius: 'var(--radius-sm)',
  cursor: 'grab',
  userSelect: 'none',
  border: '1px solid transparent',
  background: 'var(--surface-raised)',
  transition: 'all var(--duration-fast)',
};

const tileIconWrapStyle: React.CSSProperties = {
  width: 32,
  height: 32,
  borderRadius: 'var(--radius-sm)',
  background: 'var(--surface-overlay)',
  color: 'var(--accent)',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
};

const tileLabelStyle: React.CSSProperties = {
  fontSize: 10,
  color: 'var(--text-secondary)',
  textAlign: 'center',
  lineHeight: 1.3,
  maxWidth: '100%',
  wordBreak: 'break-word',
  hyphens: 'auto',
};

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------

interface SectionHeaderProps {
  icon: Icon;
  label: string;
}

function SectionHeader({ icon: Icon, label }: SectionHeaderProps) {
  return (
    <h3 style={sectionLabelStyle}>
      <Icon size={14} weight="regular" aria-hidden="true" />
      <span>{label}</span>
    </h3>
  );
}

interface TileIconProps {
  icon: ComponentType<IconProps>;
}

function TileIcon({ icon: Icon }: TileIconProps) {
  return (
    <div style={tileIconWrapStyle}>
      <Icon size={20} weight="regular" aria-hidden="true" />
    </div>
  );
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function NodePalette() {
  const [presets, setPresets] = useState<AgentPreset[]>([]);
  const mobileOpen = useUIStore((s) => s.mobilePaletteOpen);
  const closeMobile = useUIStore((s) => s.closeMobilePalette);
  const asideRef = useRef<HTMLElement | null>(null);
  const firstFocusableRef = useRef<HTMLButtonElement | null>(null);

  useEffect(() => {
    api
      .get<AgentPreset[]>('/presets')
      .then((data) => setPresets(Array.isArray(data) ? data : []))
      .catch(() => {});
  }, []);

  // Escape to close mobile palette. Coordinates with global Escape handler
  // by only firing when the drawer is actually open.
  useEffect(() => {
    if (!mobileOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        closeMobile();
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [mobileOpen, closeMobile]);

  // Simple focus trap: when mobile drawer opens, focus the close button.
  useEffect(() => {
    if (mobileOpen) {
      firstFocusableRef.current?.focus();
    }
  }, [mobileOpen]);

  const onDragStart = useCallback(
    (event: DragEvent, payload: DragPayload) => {
      setDragData(event, payload);
      // Close the mobile drawer once drag starts — desktop parity: once the
      // user starts dragging the palette is no longer needed.
      if (mobileOpen) closeMobile();
    },
    [mobileOpen, closeMobile],
  );

  const handleHover = (e: React.MouseEvent, hovering: boolean) => {
    const el = e.currentTarget as HTMLDivElement;
    el.style.borderColor = hovering ? 'var(--surface-border)' : 'transparent';
    el.style.background = hovering ? 'var(--accent-soft)' : 'var(--surface-raised)';
  };

  // When mobile-open, the aside behaves like a dialog. Desktop/tablet keep
  // the landmark-style aside semantics.
  const dialogProps = mobileOpen
    ? ({
        role: 'dialog' as const,
        'aria-modal': true as const,
        'aria-label': 'Node palette',
      })
    : {};

  return (
    <>
      <MobileHamburger />
      <SmallScreenBanner />
      <PhoneHint />

      {/* Backdrop: rendered only when mobile drawer is open. CSS hides it on
          desktop regardless, but avoiding DOM noise keeps the canvas clean. */}
      {mobileOpen && (
        <div
          className="clotho-palette-backdrop"
          onClick={closeMobile}
          aria-hidden="true"
        />
      )}

      <aside
        ref={asideRef}
        id="clotho-node-palette"
        aria-label="Node palette"
        className="clotho-palette"
        data-mobile-open={mobileOpen ? 'true' : 'false'}
        {...dialogProps}
      >
        {/* Mobile-only close button — hidden via CSS at larger breakpoints. */}
        {mobileOpen && (
          <button
            ref={firstFocusableRef}
            type="button"
            onClick={closeMobile}
            aria-label="Close node palette"
            style={{
              position: 'absolute',
              top: 10,
              right: 10,
              width: 32,
              height: 32,
              background: 'transparent',
              border: '1px solid var(--surface-border)',
              borderRadius: 'var(--radius-sm)',
              color: 'var(--text-secondary)',
              fontSize: 16,
              cursor: 'pointer',
            }}
          >
            {'\u2715'}
          </button>
        )}

        {/* ---- AGENT ---- */}
        <SectionHeader icon={Robot} label="Agent" />
        <div className="clotho-tile-grid" style={gridStyle}>
          <div
            draggable
            className="clotho-tile-primary"
            onDragStart={(e) =>
              onDragStart(e, {
                nodeType: 'agent',
                config: blankAgentConfig(),
                ports: defaultAgentPorts(),
                label: 'Prompt Agent',
              })
            }
            style={tileStyle}
            onMouseOver={(e) => handleHover(e, true)}
            onMouseOut={(e) => handleHover(e, false)}
            title="Prompt Agent"
          >
            <TileIcon icon={MagicWand} />
            <div className="clotho-tile-label" style={tileLabelStyle}>
              Prompt Agent
            </div>
          </div>
        </div>

        {/* ---- PERSONALITY ---- */}
        {presets.length > 0 && (
          <>
            <SectionHeader icon={UserCircle} label="Personality" />
            <div className="clotho-tile-grid" style={gridStyle}>
              {presets.map((preset) => {
                const Icon = presetIcon(preset.name);
                return (
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
                    data-testid={`palette-preset-${preset.name.toLowerCase().replace(/\s+/g, '-')}`}
                  >
                    <TileIcon icon={Icon} />
                    <div className="clotho-tile-label" style={tileLabelStyle}>
                      {preset.name}
                    </div>
                  </div>
                );
              })}
            </div>
          </>
        )}

        {/* ---- MEDIA ---- */}
        <SectionHeader icon={FrameCorners} label="Media" />
        <div className="clotho-tile-grid" style={gridStyle}>
          {MEDIA_ITEMS.map((item) => (
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
              <TileIcon icon={item.icon} />
              <div className="clotho-tile-label" style={tileLabelStyle}>
                {item.label}
              </div>
            </div>
          ))}
        </div>

        {/* ---- TOOLS ---- */}
        <SectionHeader icon={Wrench} label="Tools" />
        <div className="clotho-tile-grid" style={gridStyle}>
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
              <TileIcon icon={item.icon} />
              <div className="clotho-tile-label" style={tileLabelStyle}>
                {item.label}
              </div>
            </div>
          ))}
        </div>
      </aside>
    </>
  );
}
