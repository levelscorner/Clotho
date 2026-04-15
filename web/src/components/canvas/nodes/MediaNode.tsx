import React, { useCallback } from 'react';
import type { NodeProps, Node } from '@xyflow/react';
import type { MediaNodeData, MediaNodeConfig, MediaType } from '../../../lib/types';
import { BaseNode } from './BaseNode';
import { NodeRunButton } from './NodeRunButton';
import { LOCAL_MEDIA_PROVIDERS } from '../../inspector/MediaInspector';
import { usePipelineStore } from '../../../stores/pipelineStore';
import { useExecutionStore } from '../../../stores/executionStore';
import { mapError } from '../../../lib/errorRemediation';
import { resolveFileURL } from '../../../lib/api';

// ---------------------------------------------------------------------------
// Media type icons & labels
// ---------------------------------------------------------------------------

const MEDIA_ICONS: Record<MediaType, string> = {
  image: '\u{1F4F7}', // camera
  video: '\u{1F3AC}', // film clapper
  audio: '\u{1F50A}', // speaker
};

const EMPTY_COPY: Record<MediaType, string> = {
  image: 'No image produced',
  video: 'No clip produced',
  audio: 'No audio produced',
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

type MediaNodeType = Node<MediaNodeData>;

function MediaNodeInner({ id, data, selected }: NodeProps<MediaNodeType>) {
  const setSelectedNode = usePipelineStore((s) => s.setSelectedNode);
  const stepResult = useExecutionStore((s) => s.stepResults.get(id));
  const executionId = useExecutionStore((s) => s.executionId);
  const retryNode = useExecutionStore((s) => s.retryNode);

  const handleClick = useCallback(() => {
    setSelectedNode(id);
  }, [id, setSelectedNode]);

  const handleRetry = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      if (executionId) retryNode(executionId, id);
    },
    [executionId, id, retryNode],
  );

  const handleEditPrompt = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      setSelectedNode(id);
    },
    [id, setSelectedNode],
  );

  const config = data.config as MediaNodeConfig;
  const mediaType = config.media_type;
  const status = stepResult?.status;
  const rawOutput = stepResult?.output;
  // Stage B wave 5 — providers return `clotho://file/…` refs for on-disk
  // artifacts; `resolveFileURL` rewrites those to `/api/files/…` so the
  // browser can fetch them. Data URIs and external URLs pass through.
  const output = rawOutput ? resolveFileURL(rawOutput) : rawOutput;
  const error = stepResult?.error;
  const cost = stepResult?.cost;
  const duration = stepResult?.duration_ms;

  const mediaVariantClass = `clotho-node--media-${mediaType}`;
  const hasOutput = Boolean(output) && status === 'completed';
  const isEmptyComplete = status === 'completed' && !output;

  return (
    <div onClick={handleClick} role="button" tabIndex={0} onKeyDown={handleClick}>
      <BaseNode
        id={id}
        ports={data.ports}
        variant="media"
        selected={selected}
        className={mediaVariantClass}
      >
        <div className="clotho-node__header">
          <span style={{ fontSize: 16 }} aria-hidden>
            {MEDIA_ICONS[mediaType]}
          </span>
          <span className="clotho-node__label">
            {data.label}
          </span>
        </div>

        <div className="clotho-node__body">
          {/* IMAGE variant --------------------------------------------------- */}
          {mediaType === 'image' && (
            <div className="clotho-node__matte">
              <div className="clotho-node__media-preview clotho-node__media-preview--image">
                {hasOutput ? (
                  <img src={output} alt="Generated image" />
                ) : isEmptyComplete ? (
                  <span className="clotho-node__media-empty">{EMPTY_COPY.image}</span>
                ) : (
                  <span className="clotho-node__media-placeholder-label" aria-hidden>
                    IMG
                  </span>
                )}
              </div>
              <span className="clotho-node__media-readout">
                {config.provider} &middot; {config.model}
              </span>
            </div>
          )}

          {/* VIDEO variant --------------------------------------------------- */}
          {mediaType === 'video' && (
            <div className="clotho-node__reel">
              <span className="clotho-node__reel-perf clotho-node__reel-perf--top" aria-hidden />
              <div className="clotho-node__reel-frames">
                {hasOutput ? (
                  <>
                    <div className="clotho-node__reel-frame clotho-node__reel-frame--thumb" style={{ backgroundImage: `url(${output})` }} />
                    <div className="clotho-node__reel-frame clotho-node__reel-frame--thumb" style={{ backgroundImage: `url(${output})` }} />
                    <div className="clotho-node__reel-frame clotho-node__reel-frame--thumb" style={{ backgroundImage: `url(${output})` }} />
                    <div className="clotho-node__reel-frame clotho-node__reel-frame--thumb" style={{ backgroundImage: `url(${output})` }} />
                  </>
                ) : isEmptyComplete ? (
                  <span className="clotho-node__media-empty">{EMPTY_COPY.video}</span>
                ) : (
                  <>
                    <div className="clotho-node__reel-frame" />
                    <div className="clotho-node__reel-frame" />
                    <div className="clotho-node__reel-frame" />
                    <div className="clotho-node__reel-frame" />
                  </>
                )}
              </div>
              <span className="clotho-node__reel-perf clotho-node__reel-perf--bottom" aria-hidden />
              <span className="clotho-node__media-readout">
                {config.duration != null ? `${config.duration}s` : '—'}
                {' \u00B7 '}
                {config.provider}
                {' \u00B7 '}
                {status ?? 'idle'}
              </span>
            </div>
          )}

          {/* AUDIO variant --------------------------------------------------- */}
          {mediaType === 'audio' && (
            <div className="clotho-node__scope">
              {isEmptyComplete ? (
                <span className="clotho-node__media-empty">{EMPTY_COPY.audio}</span>
              ) : (
                <svg
                  className="clotho-node__scope-svg"
                  viewBox="0 0 200 60"
                  preserveAspectRatio="none"
                  aria-hidden
                >
                  <path
                    className="clotho-node__scope-path"
                    d="M0 30 Q 10 10 20 30 T 40 30 T 60 25 T 80 35 T 100 20 T 120 40 T 140 28 T 160 32 T 180 30 T 200 30"
                    stroke="var(--port-audio)"
                    strokeWidth="1.5"
                    fill="none"
                    opacity="0.85"
                  />
                </svg>
              )}
              <span className="clotho-node__media-readout">
                {config.voice ?? 'default voice'}
                {' \u00B7 '}
                {config.provider}
              </span>
            </div>
          )}
        </div>

        {/* Running: progress bar */}
        {status === 'running' && (
          <div className="clotho-node__progress-bar">
            <div className="clotho-node__progress-bar-fill" />
          </div>
        )}

        {/* Error state with recovery actions */}
        {status === 'failed' && (
          <>
            <div className="clotho-node__error">
              {mapError(error).summary}
            </div>
            <div className="clotho-node__error-actions">
              <button
                className="clotho-node__error-btn clotho-node__error-btn--primary"
                onClick={handleRetry}
              >
                Retry
              </button>
              <button
                className="clotho-node__error-btn"
                onClick={handleEditPrompt}
              >
                Edit Prompt
              </button>
            </div>
          </>
        )}

        {/* Footer with status + cost + per-node run button. Local providers
            (kokoro/comfyui/ollama) surface "local" instead of a dollar amount
            so users see zero-cost inference at a glance. Footer always
            renders so the play button is available pre-execution. */}
        <div className="clotho-node__footer">
          <span className={`clotho-node__status-dot clotho-node__status-dot--${status ?? 'idle'}`} />
          <span>{status ?? 'Idle'}</span>
          {duration != null && <span>&middot; {(duration / 1000).toFixed(1)}s</span>}
          {cost != null && (
            <span>
              &middot;{' '}
              {LOCAL_MEDIA_PROVIDERS.has(config.provider) ? (
                <span className="clotho-node__cost-local">local</span>
              ) : (
                `$${cost.toFixed(4)}`
              )}
            </span>
          )}
          <span style={{ flex: 1 }} />
          <NodeRunButton nodeId={id} />
        </div>
      </BaseNode>
    </div>
  );
}

export const MediaNode = React.memo(MediaNodeInner);
