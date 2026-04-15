import { describe, it, expect } from 'vitest';
import { describeNode } from '../nodeDescriptions';

describe('describeNode', () => {
  it('returns the generic agent description', () => {
    const desc = describeNode({ nodeType: 'agent' });
    expect(desc.teaser).toMatch(/Text LLM/);
    expect(desc.output).toBe('text');
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
});
