import { describe, it, expect } from 'vitest';
import { coerceStepFailure } from '../failureSchema';

describe('coerceStepFailure', () => {
  const valid = {
    class: 'auth',
    stage: 'provider_call',
    provider: 'openai',
    model: 'gpt-4o',
    retryable: false,
    message: 'Authentication failed',
    cause: '401 Unauthorized',
    hint: 'Verify the API key in Settings → Credentials.',
    attempts: 1,
    at: '2026-04-16T22:00:00Z',
  };

  it('accepts a fully-populated payload', () => {
    const got = coerceStepFailure(valid);
    expect(got).toEqual(valid);
  });

  it('accepts a minimal valid payload + defaults attempts/at', () => {
    const got = coerceStepFailure({
      class: 'timeout',
      stage: 'provider_call',
      message: 'Timed out',
      retryable: true,
    });
    expect(got?.class).toBe('timeout');
    expect(got?.attempts).toBe(0);
    expect(typeof got?.at).toBe('string');
  });

  it('returns undefined for non-object inputs', () => {
    expect(coerceStepFailure(null)).toBeUndefined();
    expect(coerceStepFailure(undefined)).toBeUndefined();
    expect(coerceStepFailure('failure')).toBeUndefined();
    expect(coerceStepFailure(42)).toBeUndefined();
    expect(coerceStepFailure([])).toBeUndefined();
  });

  it('rejects unknown failure class', () => {
    expect(coerceStepFailure({ ...valid, class: 'made_up' })).toBeUndefined();
  });

  it('rejects unknown stage', () => {
    expect(coerceStepFailure({ ...valid, stage: 'wrong' })).toBeUndefined();
  });

  it('rejects when message missing', () => {
    const { message: _drop, ...rest } = valid;
    expect(coerceStepFailure(rest)).toBeUndefined();
  });

  it('rejects when retryable not boolean', () => {
    expect(coerceStepFailure({ ...valid, retryable: 'true' })).toBeUndefined();
  });

  it('drops malformed optional strings without rejecting payload', () => {
    const got = coerceStepFailure({ ...valid, provider: 42, model: null });
    expect(got).toBeDefined();
    expect(got?.provider).toBeUndefined();
    expect(got?.model).toBeUndefined();
  });
});
