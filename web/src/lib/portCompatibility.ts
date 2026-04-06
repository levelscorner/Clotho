import type { PortType } from './types';

// ---------------------------------------------------------------------------
// Which target port types a given source port type can connect to.
// ---------------------------------------------------------------------------

const COMPATIBILITY: Record<PortType, readonly PortType[]> = {
  text:         ['text', 'any'],
  image_prompt: ['image_prompt', 'text', 'any'],
  video_prompt: ['video_prompt', 'text', 'any'],
  audio_prompt: ['audio_prompt', 'text', 'any'],
  image:        ['image', 'any'],
  video:        ['video', 'any'],
  audio:        ['audio', 'any'],
  json:         ['json', 'any'],
  any:          ['text', 'image_prompt', 'video_prompt', 'audio_prompt', 'image', 'video', 'audio', 'json', 'any'],
};

export function canConnect(sourceType: PortType, targetType: PortType): boolean {
  const allowed = COMPATIBILITY[sourceType];
  return allowed.includes(targetType);
}

// ---------------------------------------------------------------------------
// Visual colors for port handles
// ---------------------------------------------------------------------------

export const PORT_COLORS: Record<PortType, string> = {
  text:         '#94a3b8',
  image_prompt: '#3b82f6',
  video_prompt: '#a855f7',
  audio_prompt: '#f59e0b',
  image:        '#22c55e',
  video:        '#f97316',
  audio:        '#ec4899',
  json:         '#eab308',
  any:          '#6b7280',
};
