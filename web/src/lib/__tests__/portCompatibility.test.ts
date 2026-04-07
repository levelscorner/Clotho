import { describe, it, expect } from 'vitest';
import { canConnect } from '../portCompatibility';

describe('canConnect', () => {
  it('text connects to text', () => {
    expect(canConnect('text', 'text')).toBe(true);
  });

  it('image_prompt connects to text (subtype)', () => {
    expect(canConnect('image_prompt', 'text')).toBe(true);
  });

  it('image does not connect to text', () => {
    expect(canConnect('image', 'text')).toBe(false);
  });

  it('any connects to anything', () => {
    const allTypes = [
      'text',
      'image_prompt',
      'video_prompt',
      'audio_prompt',
      'image',
      'video',
      'audio',
      'json',
      'any',
    ] as const;

    for (const target of allTypes) {
      expect(canConnect('any', target)).toBe(true);
    }
  });

  it('video does not connect to image', () => {
    expect(canConnect('video', 'image')).toBe(false);
  });

  it('text connects to any', () => {
    expect(canConnect('text', 'any')).toBe(true);
  });
});
