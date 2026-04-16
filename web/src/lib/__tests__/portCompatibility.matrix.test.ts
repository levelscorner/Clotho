import { describe, it, expect } from 'vitest';
import { canConnect } from '../portCompatibility';
import type { PortType } from '../types';

/**
 * 81-cell port compatibility matrix. Mirror of
 * internal/domain/compatibility_matrix_test.go. Run in both surfaces so
 * the frontend (design-time validation) and backend (save + execute
 * validation) cannot drift against each other.
 *
 * The matrix is the source of truth for docs/PIPELINE-PATTERNS.md §1.
 */

const types: PortType[] = [
  'text',
  'image_prompt',
  'video_prompt',
  'audio_prompt',
  'image',
  'video',
  'audio',
  'json',
  'any',
];

const expected: Record<PortType, Record<PortType, boolean>> = {
  // Text family (text + *_prompt) is one interchangeable group.
  text: {
    text: true, image_prompt: true, video_prompt: true, audio_prompt: true,
    image: false, video: false, audio: false, json: false, any: true,
  },
  image_prompt: {
    text: true, image_prompt: true, video_prompt: true, audio_prompt: true,
    image: false, video: false, audio: false, json: false, any: true,
  },
  video_prompt: {
    text: true, image_prompt: true, video_prompt: true, audio_prompt: true,
    image: false, video: false, audio: false, json: false, any: true,
  },
  audio_prompt: {
    text: true, image_prompt: true, video_prompt: true, audio_prompt: true,
    image: false, video: false, audio: false, json: false, any: true,
  },
  image: {
    text: false, image_prompt: false, video_prompt: false, audio_prompt: false,
    image: true, video: false, audio: false, json: false, any: true,
  },
  video: {
    text: false, image_prompt: false, video_prompt: false, audio_prompt: false,
    image: false, video: true, audio: false, json: false, any: true,
  },
  audio: {
    text: false, image_prompt: false, video_prompt: false, audio_prompt: false,
    image: false, video: false, audio: true, json: false, any: true,
  },
  json: {
    text: false, image_prompt: false, video_prompt: false, audio_prompt: false,
    image: false, video: false, audio: false, json: true, any: true,
  },
  any: {
    text: true, image_prompt: true, video_prompt: true, audio_prompt: true,
    image: true, video: true, audio: true, json: true, any: true,
  },
};

describe('portCompatibility — 81-cell matrix', () => {
  it('table has 9 rows × 9 columns exactly', () => {
    expect(Object.keys(expected)).toHaveLength(9);
    for (const row of Object.values(expected)) {
      expect(Object.keys(row)).toHaveLength(9);
    }
  });

  it('every cell matches canConnect', () => {
    for (const src of types) {
      for (const tgt of types) {
        const want = expected[src][tgt];
        const got = canConnect(src, tgt);
        expect(got, `${src} → ${tgt}`).toBe(want);
      }
    }
  });

  describe('key invariants', () => {
    it('text family is fully interchangeable', () => {
      const family: PortType[] = [
        'text', 'image_prompt', 'video_prompt', 'audio_prompt',
      ];
      for (const a of family) {
        for (const b of family) {
          expect(canConnect(a, b), `${a} → ${b}`).toBe(true);
        }
      }
    });

    it('media outputs are hermetic (no cross-media)', () => {
      const media: PortType[] = ['image', 'video', 'audio'];
      for (const a of media) {
        for (const b of media) {
          if (a === b) continue;
          expect(canConnect(a, b), `${a} → ${b}`).toBe(false);
        }
      }
    });

    it('any accepts every source', () => {
      for (const src of types) {
        expect(canConnect(src, 'any'), `${src} → any`).toBe(true);
      }
    });

    it('json pairs only with itself or any', () => {
      const nonJSON: PortType[] = [
        'text', 'image_prompt', 'video_prompt',
        'audio_prompt', 'image', 'video', 'audio',
      ];
      for (const tgt of nonJSON) {
        expect(canConnect('json', tgt), `json → ${tgt}`).toBe(false);
      }
    });
  });
});
