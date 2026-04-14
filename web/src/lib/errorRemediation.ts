// ---------------------------------------------------------------------------
// errorRemediation.ts — map raw backend error strings to structured
// { summary, hint } so the UI can show a short title + an actionable
// remediation. Patterns are tried in order; the FIRST match wins.
//
// Keep the catalog small and local-provider friendly: when an OpenAI-style
// failure hits, surface a path to ComfyUI / Kokoro / Ollama where possible.
// ---------------------------------------------------------------------------

export interface RemediationEntry {
  /** Short title shown on the node body + in the inspector. */
  summary: string;
  /** Actionable next step shown in the inspector only. */
  hint: string;
}

interface CatalogEntry {
  pattern: RegExp;
  summary: string;
  hint: string;
}

// Order matters. More specific patterns should appear first.
const CATALOG: ReadonlyArray<CatalogEntry> = [
  {
    pattern: /no models pulled/i,
    summary: 'Ollama has no models installed',
    hint: 'Run `ollama pull llama3.1` in a terminal, then refresh the provider dropdown.',
  },
  {
    pattern: /rate limit/i,
    summary: 'Rate limit hit',
    hint: 'Try again in 30s, or switch to a local provider (ComfyUI / Kokoro / Ollama).',
  },
  {
    pattern: /api key|unauthorized|401/i,
    summary: 'API key missing or invalid',
    hint: 'Open the Credentials panel, add a valid key for this provider, re-run.',
  },
  {
    pattern: /timeout|context deadline/i,
    summary: 'Provider timed out',
    hint: 'Try a shorter prompt, or switch to a faster model. Kokoro responds in ~4s locally.',
  },
  {
    pattern: /not running|connection refused|ECONNREFUSED/i,
    summary: "Local server isn't running",
    hint: 'Run `make dev-full` to bring up all local services, or `ollama serve` / Kokoro / ComfyUI manually.',
  },
  {
    pattern: /invalid.*workflow|validation failed/i,
    summary: 'ComfyUI rejected the workflow',
    hint: 'Check ComfyUI logs: `make dev-logs`. Usually a missing checkpoint filename.',
  },
  {
    pattern: /model not found|404/i,
    summary: 'Model name mismatch',
    hint: 'Pick a different model from the dropdown. For Ollama: `ollama pull {model}`.',
  },
];

const FALLBACK: RemediationEntry = {
  summary: 'Step failed',
  hint: 'Check `make dev-logs` for backend output.',
};

const NO_ERROR: RemediationEntry = {
  summary: 'No error reported',
  hint: 'Nothing to remediate.',
};

/**
 * Map a raw error string to a structured remediation entry. Returns a
 * deterministic fallback when no pattern matches.
 */
export function mapError(rawError: string | null | undefined): RemediationEntry {
  if (!rawError) {
    return NO_ERROR;
  }
  for (const entry of CATALOG) {
    if (entry.pattern.test(rawError)) {
      return { summary: entry.summary, hint: entry.hint };
    }
  }
  return FALLBACK;
}
