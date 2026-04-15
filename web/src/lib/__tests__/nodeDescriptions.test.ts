import { describe, it, expect } from 'vitest';
import { describeNode } from '../nodeDescriptions';

describe('describeNode', () => {
  it('falls back to generic agent description when no preset is provided', () => {
    const desc = describeNode({ nodeType: 'agent' });
    expect(desc.teaser).toMatch(/Text LLM/);
    expect(desc.output).toBe('text');
  });

  it('resolves agent:script when preset_category is "script"', () => {
    const desc = describeNode({ nodeType: 'agent', presetCategory: 'script' });
    expect(desc.teaser).toMatch(/narrative/i);
    expect(desc.output).toMatch(/script/);
  });

  it('resolves media:image when media_type is "image"', () => {
    const desc = describeNode({ nodeType: 'media', mediaType: 'image' });
    expect(desc.teaser).toMatch(/image/i);
    expect(desc.output).toMatch(/PNG/);
  });

  it('resolves tool:text_box when tool_type is "text_box"', () => {
    const desc = describeNode({ nodeType: 'tool', toolType: 'text_box' });
    expect(desc.teaser).toMatch(/Static text/);
    expect(desc.input).toBe('(none)');
    expect(desc.output).toBe('text');
  });

  it('prefers presetDescription over static dictionary when provided', () => {
    const custom = 'Writes haikus in the voice of a seagull';
    const desc = describeNode({
      nodeType: 'agent',
      presetCategory: 'script',
      presetDescription: custom,
    });
    expect(desc.full).toBe(custom);
    expect(desc.teaser).toBe(custom);
    // Input/output fall back to the generic agent defaults.
    expect(desc.output).toBe('text');
  });
});
