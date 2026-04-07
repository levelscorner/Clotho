import React, { useCallback } from 'react';
import type { NodeProps, Node } from '@xyflow/react';
import type { MediaNodeData, MediaNodeConfig, MediaType } from '../../../lib/types';
import { BaseNode } from './BaseNode';
import { usePipelineStore } from '../../../stores/pipelineStore';
import { useExecutionStore } from '../../../stores/executionStore';

// ---------------------------------------------------------------------------
// Media type icons & labels
// ---------------------------------------------------------------------------

const MEDIA_ICONS: Record<MediaType, string> = {
  image: '\u{1F4F7}', // camera
  video: '\u{1F3AC}', // film clapper
  audio: '\u{1F50A}', // speaker
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
  const output = stepResult?.output;
  const error = stepResult?.error;
  const cost = stepResult?.cost;
  const duration = stepResult?.duration_ms;

  const mediaVariantClass = `clotho-node--media-${mediaType}`;

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
          <span className="clotho-node__badge">{config.provider}</span>{' '}
          <span className="clotho-node__badge">{config.model}</span>
        </div>

        {/* Running: progress bar */}
        {status === 'running' && (
          <div className="clotho-node__progress-bar">
            <div className="clotho-node__progress-bar-fill" />
          </div>
        )}

        {/* Completed: media preview */}
        {output && status === 'completed' && mediaType === 'image' && (
          <div className="clotho-node__media-preview">
            <img src={output} alt="Generated image" />
          </div>
        )}

        {output && status === 'completed' && mediaType === 'video' && (
          <div className="clotho-node__media-preview clotho-node__media-preview--video">
            <img src={output} alt="Video thumbnail" />
            <div className="clotho-node__media-play-overlay">
              {'\u25B6'}
            </div>
          </div>
        )}

        {output && status === 'completed' && mediaType === 'audio' && (
          <div className="clotho-node__media-audio-placeholder">
            <span style={{ fontSize: 14 }} aria-hidden>
              {'\u25B6'}
            </span>
            <span>Audio ready</span>
          </div>
        )}

        {/* Error state with recovery actions */}
        {status === 'failed' && (
          <>
            <div className="clotho-node__error">
              {error || 'Execution failed'}
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

        {/* Footer with status + cost */}
        {status && (
          <div className="clotho-node__footer">
            <span className={`clotho-node__status-dot clotho-node__status-dot--${status}`} />
            <span>{status}</span>
            {duration != null && <span>&middot; {(duration / 1000).toFixed(1)}s</span>}
            {cost != null && <span>&middot; ${cost.toFixed(4)}</span>}
          </div>
        )}
      </BaseNode>
    </div>
  );
}

export const MediaNode = React.memo(MediaNodeInner);
