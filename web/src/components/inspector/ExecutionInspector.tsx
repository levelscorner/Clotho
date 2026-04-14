import type { StepResult, ExecutionStatus } from '../../lib/types';
import { InspectorGroup } from './InspectorGroup';
import { mapError } from '../../lib/errorRemediation';
import { resolveFileURL, isResolvableFile, revealInFinder } from '../../lib/api';

// ---------------------------------------------------------------------------
// File-URL extension sniffing — used when the step output is a
// `clotho://file/...` reference. The backend may hand back .png, .mp4, .mp3,
// or an unknown extension; we match against the actual filename so we pick
// the right media element without round-tripping to the server.
// ---------------------------------------------------------------------------

const IMAGE_EXTS = ['.png', '.jpg', '.jpeg', '.webp', '.gif', '.bmp'];
const AUDIO_EXTS = ['.mp3', '.m4a', '.wav', '.ogg', '.flac'];
const VIDEO_EXTS = ['.mp4', '.webm', '.mov'];

type FileMediaKind = 'image' | 'audio' | 'video' | 'other';

function classifyFileRef(ref: string): FileMediaKind {
  const lower = ref.toLowerCase();
  if (IMAGE_EXTS.some((ext) => lower.endsWith(ext))) return 'image';
  if (AUDIO_EXTS.some((ext) => lower.endsWith(ext))) return 'audio';
  if (VIDEO_EXTS.some((ext) => lower.endsWith(ext))) return 'video';
  return 'other';
}

function filenameFromRef(ref: string): string {
  const slash = ref.lastIndexOf('/');
  return slash >= 0 ? ref.slice(slash + 1) : ref;
}

// ---------------------------------------------------------------------------
// Status badge colours
// ---------------------------------------------------------------------------

const STATUS_COLORS: Record<ExecutionStatus, string> = {
  pending: '#64748b',
  running: '#3b82f6',
  completed: '#22c55e',
  failed: '#ef4444',
  cancelled: '#f59e0b',
  skipped: '#6b7280',
};

// ---------------------------------------------------------------------------
// Byte formatter — turns output.length into "1.1 MB" / "12 KB" / "320 B".
// ---------------------------------------------------------------------------

const KB = 1024;
const MB = KB * 1024;

function formatBytes(n: number): string {
  if (n < KB) return `${n} B`;
  if (n < MB) return `${(n / KB).toFixed(1)} KB`;
  return `${(n / MB).toFixed(1)} MB`;
}

// Only show a size hint when the output is meaningfully large (>= 1 KB) —
// a 200 B prompt answer doesn't need a size annotation in the title.
function outputSizeHint(output: string | undefined): string {
  if (!output || output.length < KB) return '';
  return ` (${formatBytes(output.length)})`;
}

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface ExecutionInspectorProps {
  step: StepResult;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function ExecutionInspector({ step }: ExecutionInspectorProps) {
  const hasDetails = Boolean(step.output || step.error);
  const sizeHint = outputSizeHint(step.output);
  const remediation = step.error ? mapError(step.error) : null;

  return (
    <div style={{ fontSize: 13 }}>
      <InspectorGroup title="Summary" defaultOpen>
        <div style={{ marginBottom: 10, display: 'flex', alignItems: 'center', gap: 8 }}>
          <span
            style={{
              display: 'inline-block',
              padding: '2px 8px',
              borderRadius: 4,
              fontSize: 11,
              fontWeight: 600,
              background: STATUS_COLORS[step.status],
              color: '#fff',
            }}
          >
            {step.status}
          </span>
        </div>

        <div
          style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: 8,
            fontSize: 12,
            color: '#94a3b8',
          }}
        >
          {step.tokens_used != null && (
            <div>Tokens: {step.tokens_used.toLocaleString()}</div>
          )}
          {step.cost != null && <div>Cost: ${step.cost.toFixed(4)}</div>}
          {step.duration_ms != null && (
            <div>Duration: {(step.duration_ms / 1000).toFixed(1)}s</div>
          )}
        </div>
      </InspectorGroup>

      {hasDetails && (
        <InspectorGroup title="Details" forceOpen={Boolean(step.error)}>
          {step.output && (
            <div style={{ marginBottom: 10 }}>
              {/* A5 — the Output group defaults closed so a 1 MB base64 blob
                  doesn't flatten the inspector. Errors auto-expand it. */}
              <InspectorGroup
                title={`Model Output${sizeHint}`}
                defaultOpen={false}
                forceOpen={Boolean(step.error)}
              >
                <OutputPreview output={step.output} />
                {isResolvableFile(step.output) && (
                  <div style={{ marginTop: 8 }}>
                    <button
                      type="button"
                      className="clotho-reveal-btn"
                      onClick={() => revealInFinder(step.output!)}
                      title="Open containing folder"
                    >
                      Reveal in folder
                    </button>
                  </div>
                )}
              </InspectorGroup>
            </div>
          )}

          {step.error && remediation && (
            <div
              role="alert"
              style={{
                marginBottom: 10,
                padding: 10,
                borderRadius: 4,
                background: 'rgba(127, 29, 29, 0.25)',
                border: '1px solid #7f1d1d',
              }}
            >
              <div className="clotho-error-summary">{remediation.summary}</div>
              <div className="clotho-error-hint">{remediation.hint}</div>
              <details style={{ marginTop: 8 }}>
                <summary
                  style={{
                    cursor: 'pointer',
                    fontSize: 11,
                    color: '#94a3b8',
                    textTransform: 'uppercase',
                    letterSpacing: 0.3,
                  }}
                >
                  Raw error
                </summary>
                <pre
                  style={{
                    marginTop: 6,
                    background: '#1a1c2e',
                    padding: 8,
                    borderRadius: 4,
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                    color: '#fca5a5',
                    fontSize: 12,
                    border: '1px solid #7f1d1d',
                    maxHeight: 240,
                    overflow: 'auto',
                  }}
                >
                  {step.error}
                </pre>
              </details>
            </div>
          )}
        </InspectorGroup>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Output preview — renders the step output with media-type awareness.
//
// Image/audio/video data URIs become inline <img>/<audio>/<video> elements
// so the inspector shows a real preview instead of a multi-megabyte base64
// string. Everything else falls back to a scrollable <pre>.
// ---------------------------------------------------------------------------

function OutputPreview({ output }: { output: string }) {
  const mediaBlock: React.CSSProperties = {
    background: '#1a1c2e',
    padding: 8,
    borderRadius: 4,
    border: '1px solid #334155',
    maxHeight: 320,
  };

  // Stage B wave 5 — clotho://file/ references are on-disk artifacts written
  // by the provider layer. We resolve them to /api/files/{path} for the
  // <img> / <audio> / <video> src, and we sniff the extension to pick the
  // right element. Anything we can't classify falls back to a text link so
  // users can still reach the artifact.
  if (isResolvableFile(output)) {
    const resolved = resolveFileURL(output);
    const kind = classifyFileRef(output);
    const filename = filenameFromRef(output);
    if (kind === 'image') {
      return (
        <div style={mediaBlock}>
          <img
            src={resolved}
            alt="Generated output"
            style={{ width: '100%', height: 'auto', borderRadius: 3, display: 'block' }}
          />
        </div>
      );
    }
    if (kind === 'audio') {
      return (
        <div style={mediaBlock}>
          <audio controls src={resolved} style={{ width: '100%' }} />
        </div>
      );
    }
    if (kind === 'video') {
      return (
        <div style={mediaBlock}>
          <video controls src={resolved} style={{ width: '100%', borderRadius: 3 }} />
        </div>
      );
    }
    return (
      <a
        href={resolved}
        target="_blank"
        rel="noopener noreferrer"
        style={{
          display: 'inline-block',
          padding: '6px 10px',
          background: '#1a1c2e',
          border: '1px solid #334155',
          borderRadius: 4,
          color: '#e2e8f0',
          fontSize: 12,
          textDecoration: 'none',
        }}
      >
        Open {filename}
      </a>
    );
  }

  if (output.startsWith('data:image/')) {
    return (
      <div style={mediaBlock}>
        <img
          src={output}
          alt="Generated output"
          style={{ width: '100%', height: 'auto', borderRadius: 3, display: 'block' }}
        />
      </div>
    );
  }
  if (output.startsWith('data:audio/')) {
    return (
      <div style={mediaBlock}>
        <audio controls src={output} style={{ width: '100%' }} />
      </div>
    );
  }
  if (output.startsWith('data:video/')) {
    return (
      <div style={mediaBlock}>
        <video controls src={output} style={{ width: '100%', borderRadius: 3 }} />
      </div>
    );
  }
  // Truncate very long plain-text outputs so we never paint a 1MB <pre>.
  const MAX = 8000;
  const shown = output.length > MAX ? output.slice(0, MAX) + '\n…(truncated)' : output;
  return (
    <pre
      style={{
        background: '#1a1c2e',
        padding: 8,
        borderRadius: 4,
        whiteSpace: 'pre-wrap',
        wordBreak: 'break-word',
        maxHeight: 300,
        overflow: 'auto',
        fontSize: 12,
        color: '#e2e8f0',
        border: '1px solid #334155',
      }}
    >
      {shown}
    </pre>
  );
}
