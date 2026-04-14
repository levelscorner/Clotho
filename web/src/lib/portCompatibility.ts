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
  text:         'var(--port-text)',
  image_prompt: 'var(--port-image)',
  video_prompt: 'var(--port-video)',
  audio_prompt: 'var(--port-audio)',
  image:        'var(--port-image)',
  video:        'var(--port-video)',
  audio:        'var(--port-audio)',
  json:         'var(--accent)',
  any:          'var(--text-muted)',
};

// ---------------------------------------------------------------------------
// Human-readable labels for port types. Used by the hover-reveal port labels
// on nodes so users see "image prompt" instead of "image_prompt" at a glance.
// ---------------------------------------------------------------------------

export const PORT_TYPE_LABEL: Record<PortType, string> = {
  text:         'text',
  image_prompt: 'image prompt',
  video_prompt: 'video prompt',
  audio_prompt: 'audio prompt',
  image:        'image',
  video:        'video',
  audio:        'audio',
  json:         'json',
  any:          'any',
};
