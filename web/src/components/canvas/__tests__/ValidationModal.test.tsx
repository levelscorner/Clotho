import { describe, it, expect } from 'vitest';
import { parseValidationErrors } from '../ValidationModal';

describe('parseValidationErrors', () => {
  it('parses backend 400 with validation_errors', () => {
    const err = new Error(
      JSON.stringify({
        error: 'graph validation failed',
        validation_errors: [
          { field: 'nodes', message: 'duplicate node ID: foo' },
          { field: 'edges[e1].source_port', message: 'source port "x" does not exist' },
        ],
      }),
    );
    const got = parseValidationErrors(err);
    expect(got).not.toBeNull();
    expect(got?.length).toBe(2);
    expect(got?.[0].message).toContain('duplicate node ID');
    expect(got?.[1].field).toBe('edges[e1].source_port');
  });

  it('returns null for non-validation errors', () => {
    expect(parseValidationErrors(new Error('Internal Server Error'))).toBeNull();
    expect(parseValidationErrors(null)).toBeNull();
    expect(parseValidationErrors(undefined)).toBeNull();
    expect(parseValidationErrors({ message: 'plain string' })).toBeNull();
  });

  it('returns null for valid JSON without validation_errors', () => {
    const err = new Error(JSON.stringify({ error: 'forbidden' }));
    expect(parseValidationErrors(err)).toBeNull();
  });

  it('drops malformed entries inside validation_errors', () => {
    const err = new Error(
      JSON.stringify({
        validation_errors: [
          { field: 'a', message: 'good' },
          { field: 42, message: null }, // wrong types — drop
          { field: 'b', message: 'also good' },
        ],
      }),
    );
    const got = parseValidationErrors(err);
    expect(got?.length).toBe(2);
    expect(got?.[0].message).toBe('good');
    expect(got?.[1].message).toBe('also good');
  });
});
