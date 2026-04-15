// ---------------------------------------------------------------------------
// Preset icon map
// ---------------------------------------------------------------------------
// Maps a preset name (lowercased) → a Phosphor icon component. The sidebar
// palette uses this to render Personality tiles with a meaningful glyph
// instead of an initial-based chip. Unknown names fall back to the category
// default (`UserCircle`).
//
// We keep the map client-side so existing preset rows in the database keep
// working without a migration; new presets can also ship their own `icon`
// field in future.
// ---------------------------------------------------------------------------

import {
  PenNib,
  Scroll,
  UsersThree,
  Palette,
  FilmSlate,
  ArrowsClockwise,
  Sparkle,
  UserCircle,
} from 'phosphor-react';
import type { Icon } from 'phosphor-react';

export const PRESET_ICONS: Record<string, Icon> = {
  'script writer': PenNib,
  'story writer': Scroll,
  'character designer': UsersThree,
  'image prompt crafter': Palette,
  'video prompt writer': FilmSlate,
  'story-to-prompt': ArrowsClockwise,
  'prompt enhancer': Sparkle,
};

export function presetIcon(name: string): Icon {
  return PRESET_ICONS[name.trim().toLowerCase()] ?? UserCircle;
}
