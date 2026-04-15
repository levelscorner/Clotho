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

  it('renders all 4 section headers', async () => {
    render(<NodePalette />);
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /agent/i })).toBeInTheDocument();
    });
    expect(screen.getByRole('heading', { name: /personality/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /media/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /tools/i })).toBeInTheDocument();
  });

  it('renders Prompt Agent tile under Agent', () => {
    render(<NodePalette />);
    expect(screen.getByText('Prompt Agent')).toBeInTheDocument();
    expect(screen.queryByText('New Agent')).not.toBeInTheDocument();
  });

  it('renders all media tiles with their labels', () => {
    render(<NodePalette />);
    expect(screen.getByText('Image Gen')).toBeInTheDocument();
    expect(screen.getByText('Video Gen')).toBeInTheDocument();
    expect(screen.getByText('Voice TTS')).toBeInTheDocument();
  });

  it('renders all tool tiles with their labels', () => {
    render(<NodePalette />);
    expect(screen.getByText('Text')).toBeInTheDocument();
    expect(screen.getByText('Image')).toBeInTheDocument();
    expect(screen.getByText('Video')).toBeInTheDocument();
  });

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
