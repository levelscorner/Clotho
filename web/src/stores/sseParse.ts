/**
 * Defensive parser for SSE event envelopes streamed by the Go engine.
 *
 * The previous code did `JSON.parse(e.data) as EventEnvelope<…>` and then
 * reached into fields without validating their shape. A malformed or
 * attacker-supplied payload could reach node state with unexpected types —
 * not catastrophic here, but a regression risk.
 *
 * This module exposes a single `parseEnvelope` that returns a typed
 * result on success and `null` on any structural mismatch. Callers bail
 * out on null instead of passing garbage into the store.
 *
 * We don't pull in Zod — the envelope is tiny, the validator is
 * ~30 lines, and runtime bundle cost matters for the extension-free web
 * bundle.
 */

export interface BaseEnvelope {
  type?: string;
  execution_id?: string;
  node_id?: string;
  timestamp?: string;
  error?: string;
}

/**
 * Parse a JSON string into a validated envelope with the payload typed as
 * `T`. Returns `null` when the input is not an object, is unparseable, or
 * its top-level fields are the wrong shape. Payload fields (`data`) are
 * returned as-is — callers that need deeper validation do it inline at
 * the use site.
 */
export function parseEnvelope<T = unknown>(
  raw: string,
): (BaseEnvelope & { data?: T }) | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return null;
  }
  const obj = parsed as Record<string, unknown>;

  // Allow missing top-level fields — some event types legitimately
  // omit node_id (execution_completed) or data (step_started). Only
  // reject when a field is present and has the wrong type.
  if ('type' in obj && obj.type !== undefined && typeof obj.type !== 'string') return null;
  if ('execution_id' in obj && obj.execution_id !== undefined && typeof obj.execution_id !== 'string') return null;
  if ('node_id' in obj && obj.node_id !== undefined && typeof obj.node_id !== 'string') return null;
  if ('timestamp' in obj && obj.timestamp !== undefined && typeof obj.timestamp !== 'string') return null;
  if ('error' in obj && obj.error !== undefined && typeof obj.error !== 'string') return null;

  return obj as BaseEnvelope & { data?: T };
}
