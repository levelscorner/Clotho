import React, { useCallback } from 'react';
import { FolderOpen } from 'phosphor-react';
import { useExecutionStore } from '../../../stores/executionStore';
import { revealInFinder, isResolvableFile } from '../../../lib/api';

// ---------------------------------------------------------------------------
// NodeFolderButton
//
// Small folder icon in the node footer. Visible only when the node has a
// resolvable on-disk artifact — agent `.txt` files, or media `image-*.png`
// / `audio-*.mp3` / `video-*.mp4` assets. Tools don't produce files, so
// the button stays hidden for them.
//
// Click reveals the file in the OS file manager via the existing
// POST /api/files/reveal endpoint. On macOS that's `open -R {abs}`, which
// opens the parent folder with the file selected.
// ---------------------------------------------------------------------------

interface NodeFolderButtonProps {
  nodeId: string;
}

function NodeFolderButtonInner({ nodeId }: NodeFolderButtonProps) {
  const stepResult = useExecutionStore((s) => s.stepResults.get(nodeId));

  // Prefer the explicit output_file field (populated by the engine for
  // both agent and media nodes). Fall back to the output when it happens
  // to be a clotho://file/ ref — covers legacy executions where
  // output_file wasn't plumbed yet.
  const fileRef =
    stepResult?.output_file ??
    (stepResult?.output && isResolvableFile(stepResult.output)
      ? stepResult.output
      : undefined);

  const handleClick = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      if (!fileRef) return;
      await revealInFinder(fileRef);
    },
    [fileRef],
  );

  if (!fileRef) return null;

  return (
    <button
      type="button"
      className="clotho-node__folder"
      onClick={handleClick}
      onMouseDown={(e) => e.stopPropagation()}
      aria-label="Open output in folder"
      title="Open output in folder"
    >
      <FolderOpen size={12} weight="regular" aria-hidden="true" />
    </button>
  );
}

export const NodeFolderButton = React.memo(NodeFolderButtonInner);
