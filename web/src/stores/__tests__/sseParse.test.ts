import { describe, it, expect } from 'vitest';
import { parseEnvelope } from '../sseParse';

describe('parseEnvelope', () => {
  it('returns null for invalid JSON', () => {
    expect(parseEnvelope('not json')).toBeNull();
    expect(parseEnvelope('')).toBeNull();
    expect(parseEnvelope('null')).toBeNull();
    expect(parseEnvelope('[]')).toBeNull();
    expect(parseEnvelope('42')).toBeNull();
    expect(parseEnvelope('"a string"')).toBeNull();
  });

  it('rejects wrong-typed top-level fields', () => {
    expect(parseEnvelope('{"type": 42}')).toBeNull();
    expect(parseEnvelope('{"node_id": true}')).toBeNull();
    expect(parseEnvelope('{"timestamp": {}}')).toBeNull();
    expect(parseEnvelope('{"error": []}')).toBeNull();
  });

  it('accepts a minimal valid envelope', () => {
    const env = parseEnvelope('{"type": "step_started", "node_id": "n1"}');
    expect(env).not.toBeNull();
    expect(env?.type).toBe('step_started');
    expect(env?.node_id).toBe('n1');
  });

  it('accepts envelopes with arbitrary data payload', () => {
    const env = parseEnvelope<{ chunk: string }>(
      '{"type": "step_chunk", "node_id": "n1", "data": {"chunk": "hello"}}',
    );
    expect(env?.data?.chunk).toBe('hello');
  });

  it('accepts envelopes that omit optional fields', () => {
    const env = parseEnvelope('{"type": "execution_completed"}');
    expect(env).not.toBeNull();
    expect(env?.type).toBe('execution_completed');
    expect(env?.node_id).toBeUndefined();
  });
});
