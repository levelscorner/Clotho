import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import { NodePalette } from '../NodePalette';
import { useUIStore } from '../../../stores/uiStore';

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
    // Force the "show all sections" branch. On desktop the palette is a
    // click-to-open flyout (only one section at a time); mobilePaletteOpen
    // toggles the legacy full-drawer view, which surfaces all sections so
    // structural assertions below still have something to inspect.
    useUIStore.setState({ mobilePaletteOpen: true });
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
});
