import type { FailureClass, FailureStage, StepFailure } from './types';

// Source of truth for the FailureClass union — kept here as a runtime
// Set so coerceStepFailure can validate. Mirrors internal/domain/failure.go
// and the FailureClass union in ./types.ts.
const KNOWN_CLASSES: ReadonlySet<FailureClass> = new Set<FailureClass>([
  'network',
  'rate_limit',
  'timeout',
  'auth',
  'provider_5xx',
  'provider_4xx',
  'validation',
  'output_shape',
  'output_quality',
  'cost_cap',
  'circuit_open',
  'internal',
]);

const KNOWN_STAGES: ReadonlySet<FailureStage> = new Set<FailureStage>([
  'input_resolve',
  'provider_call',
  'stream_parse',
  'output_validate',
  'persist',
]);

/**
 * Defensive coercion of an unknown SSE payload field into a typed
 * StepFailure. Returns undefined when the payload is missing, malformed,
 * or carries an unknown class/stage — never throws and never returns
 * partial garbage. Callers fall back to the legacy `error` string when
 * undefined.
 *
 * Anchored to runtime checks rather than `as` casts because SSE wire
 * data is ultimately attacker-controllable and the FailureDrawer renders
 * Cause / Hint as user-visible text.
 */
export function coerceStepFailure(input: unknown): StepFailure | undefined {
  if (!input || typeof input !== 'object' || Array.isArray(input)) {
    return undefined;
  }
  const obj = input as Record<string, unknown>;

  if (typeof obj.class !== 'string' || !KNOWN_CLASSES.has(obj.class as FailureClass)) {
    return undefined;
  }
  if (typeof obj.stage !== 'string' || !KNOWN_STAGES.has(obj.stage as FailureStage)) {
    return undefined;
  }
  if (typeof obj.message !== 'string') {
    return undefined;
  }
  if (typeof obj.retryable !== 'boolean') {
    return undefined;
  }

  // Optional fields — coerce or drop. Never let an unexpected type
  // through; better to omit a field than to render `[object Object]`.
  const optStr = (k: string): string | undefined =>
    typeof obj[k] === 'string' ? (obj[k] as string) : undefined;
  const attempts =
    typeof obj.attempts === 'number' && Number.isFinite(obj.attempts)
      ? obj.attempts
      : 0;
  const at = typeof obj.at === 'string' ? obj.at : new Date().toISOString();

  return {
    class: obj.class as FailureClass,
    stage: obj.stage as FailureStage,
    provider: optStr('provider'),
    model: optStr('model'),
    retryable: obj.retryable,
    message: obj.message,
    cause: optStr('cause'),
    hint: optStr('hint'),
    attempts,
    at,
  };
}
