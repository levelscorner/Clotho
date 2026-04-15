import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import { NodePalette } from '../NodePalette';
import { api } from '../../../lib/api';
import type { AgentPreset } from '../../../lib/types';

// ---------------------------------------------------------------------------
// Test fixtures — one preset per personality so we can assert icon mapping
// works for known names and falls back gracefully for unknown ones.
// ---------------------------------------------------------------------------

const PRESET_FIXTURES: AgentPreset[] = [
  {
    id: 'p1',
    name: 'Script Writer',
    description: '',
    category: 'script',
    icon: '',
    is_built_in: true,
    config: {
      provider: 'openai',
      model: 'gpt-4o',
      role: { system_prompt: '', persona: '' },
      task: { task_type: 'custom', output_type: 'text', template: '' },
      temperature: 0.7,
      max_tokens: 2048,
    },
  },
  {
    id: 'p2',
    name: 'Image Prompt Crafter',
    description: '',
    category: 'image',
    icon: '',
    is_built_in: true,
    config: {
      provider: 'openai',
      model: 'gpt-4o',
      role: { system_prompt: '', persona: '' },
      task: { task_type: 'custom', output_type: 'image_prompt', template: '' },
      temperature: 0.7,
      max_tokens: 2048,
    },
  },
  {
    id: 'p3',
    name: 'Mystery Unknown Preset',
    description: '',
    category: 'other',
    icon: '',
    is_built_in: false,
    config: {
      provider: 'openai',
      model: 'gpt-4o',
      role: { system_prompt: '', persona: '' },
      task: { task_type: 'custom', output_type: 'text', template: '' },
      temperature: 0.7,
      max_tokens: 2048,
    },
  },
];

describe('NodePalette', () => {
  beforeEach(() => {
    // jsdom doesn't implement matchMedia; PhoneHint uses it on mount.
    if (!window.matchMedia) {
      Object.defineProperty(window, 'matchMedia', {
        writable: true,
        value: vi.fn().mockImplementation((query: string) => ({
          matches: false,
          media: query,
          onchange: null,
          addListener: vi.fn(),
          removeListener: vi.fn(),
          addEventListener: vi.fn(),
          removeEventListener: vi.fn(),
          dispatchEvent: vi.fn(),
        })),
      });
    }
    vi.spyOn(api, 'get').mockResolvedValue(PRESET_FIXTURES as never);
  });

  // -----------------------------------------------------------------------
  // Structure: three sections only — Agent / Personality / Tools.
  // "Media" used to be a fourth section; it has been collapsed into Agent.
  // -----------------------------------------------------------------------

  it('renders exactly three section headers: Agent, Personality, Tools', async () => {
    render(<NodePalette />);
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /agent/i })).toBeInTheDocument();
    });
    expect(screen.getByRole('heading', { name: /personality/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /tools/i })).toBeInTheDocument();
    // Media header should no longer exist anywhere in the palette.
    expect(screen.queryByRole('heading', { name: /^media$/i })).not.toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Agent section — four modality tiles with locked standard icons.
  // We scope lookups by data-testid to avoid label collisions with Tools
  // (which also has "Image" and "Video" tiles).
  // -----------------------------------------------------------------------

  it('renders all 4 Agent tiles: Prompt / Image / Audio / Video', () => {
    render(<NodePalette />);
    expect(screen.getByTestId('palette-agent-prompt')).toBeInTheDocument();
    expect(screen.getByTestId('palette-agent-image')).toBeInTheDocument();
    expect(screen.getByTestId('palette-agent-audio')).toBeInTheDocument();
    expect(screen.getByTestId('palette-agent-video')).toBeInTheDocument();
  });

  it('Agent tiles show their scoped labels', () => {
    render(<NodePalette />);
    expect(
      within(screen.getByTestId('palette-agent-prompt')).getByText('Prompt'),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId('palette-agent-image')).getByText('Image'),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId('palette-agent-audio')).getByText('Audio'),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId('palette-agent-video')).getByText('Video'),
    ).toBeInTheDocument();
  });

  it('Agent tiles render SVG icons (MagicWand/ImageSquare/SpeakerHigh/VideoCamera)', () => {
    render(<NodePalette />);
    for (const kind of ['prompt', 'image', 'audio', 'video'] as const) {
      const tile = screen.getByTestId(`palette-agent-${kind}`);
      const svg = tile.querySelector('svg');
      expect(svg, `expected svg icon inside palette-agent-${kind}`).not.toBeNull();
    }
  });

  it('Prompt tile drag payload creates an agent node (not a media node)', () => {
    render(<NodePalette />);
    const tile = screen.getByTestId('palette-agent-prompt');
    const dragEvent = new Event('dragstart', { bubbles: true }) as unknown as DragEvent;
    let captured = '';
    Object.defineProperty(dragEvent, 'dataTransfer', {
      value: {
        setData: (_type: string, data: string) => {
          captured = data;
        },
        effectAllowed: '',
      },
    });
    tile.dispatchEvent(dragEvent);
    const parsed = JSON.parse(captured);
    expect(parsed.nodeType).toBe('agent');
    expect(parsed.label).toBe('Prompt Agent');
  });

  it('Agent Image/Audio/Video tiles drag media node payloads with correct defaults', () => {
    render(<NodePalette />);
    const cases: Array<{
      kind: 'image' | 'audio' | 'video';
      provider: string;
      model: string;
      voice?: string;
    }> = [
      { kind: 'image', provider: 'comfyui', model: 'flux1-schnell' },
      { kind: 'audio', provider: 'kokoro', model: 'kokoro', voice: 'af_bella' },
      { kind: 'video', provider: 'replicate', model: 'stable-video-diffusion' },
    ];

    for (const { kind, provider, model, voice } of cases) {
      const tile = screen.getByTestId(`palette-agent-${kind}`);
      const dragEvent = new Event('dragstart', { bubbles: true }) as unknown as DragEvent;
      let captured = '';
      Object.defineProperty(dragEvent, 'dataTransfer', {
        value: {
          setData: (_type: string, data: string) => {
            captured = data;
          },
          effectAllowed: '',
        },
      });
      tile.dispatchEvent(dragEvent);
      const parsed = JSON.parse(captured);
      expect(parsed.nodeType, `${kind} tile nodeType`).toBe('media');
      expect(parsed.config.media_type, `${kind} media_type`).toBe(kind);
      expect(parsed.config.provider, `${kind} provider`).toBe(provider);
      expect(parsed.config.model, `${kind} model`).toBe(model);
      if (voice) {
        expect(parsed.config.voice, `${kind} voice`).toBe(voice);
      }
    }
  });

  // -----------------------------------------------------------------------
  // Tools section — unchanged.
  // -----------------------------------------------------------------------

  it('renders all tool tiles with their labels', () => {
    render(<NodePalette />);
    // Both Agent and Tools have Image + Video labels; assert at least 2 of each.
    expect(screen.getAllByText('Image').length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByText('Video').length).toBeGreaterThanOrEqual(2);
    // "Text" label only appears in the Tools section.
    expect(screen.getByText('Text')).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Personality section — unchanged (presetIcons.ts lookup).
  // -----------------------------------------------------------------------

  it('renders preset tiles with Phosphor SVG icons (no 2-letter chip)', async () => {
    render(<NodePalette />);
    await waitFor(() => {
      expect(screen.getByText('Script Writer')).toBeInTheDocument();
    });

    const scriptTile = screen.getByTestId('palette-preset-script-writer');
    // Phosphor icons render as <svg>. Presence of an SVG inside the tile
    // proves the chip was replaced.
    const svg = scriptTile.querySelector('svg');
    expect(svg).not.toBeNull();

    // Legacy initial chip ("Sw") must not exist anywhere.
    expect(within(scriptTile).queryByText('Sw')).toBeNull();
  });

  it('falls back to default icon for unknown preset names without crashing', async () => {
    render(<NodePalette />);
    await waitFor(() => {
      expect(screen.getByText('Mystery Unknown Preset')).toBeInTheDocument();
    });
    const unknownTile = screen.getByTestId(
      'palette-preset-mystery-unknown-preset',
    );
    const svg = unknownTile.querySelector('svg');
    expect(svg).not.toBeNull();
  });
});
