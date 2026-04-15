import { useCallback } from 'react';
import { FolderOpen } from 'phosphor-react';
import { useExecutionStore } from '../../stores/executionStore';
import { revealInFinder } from '../../lib/api';

/**
 * Top-bar button that reveals the current execution's output folder
 * in Finder / Explorer / xdg-open. Appears only after an execution
 * completes and sets `artifactDir` — hidden during streaming and
 * before any run.
 *
 * The path comes from the execution_completed SSE event's
 * artifact_dir field, which the engine builds via storage.RelDir(loc)
 * so it always matches where the files actually landed.
 */
export function OpenFolderButton() {
  const artifactDir = useExecutionStore((s) => s.artifactDir);

  const handleOpen = useCallback(async () => {
    if (!artifactDir) return;
    // revealInFinder accepts both a clotho://file/ URL and a bare relative
    // path — the /api/files/reveal endpoint validates either form. We pass
    // the dir-with-trailing-slash form so macOS selects the folder itself.
    await revealInFinder(`clotho://file/${artifactDir}`);
  }, [artifactDir]);

  if (!artifactDir) return null;

  return (
    <button
      type="button"
      onClick={handleOpen}
      title={`Open the output folder: ${artifactDir}`}
      style={{
        padding: '6px 10px',
        minHeight: 32,
        borderRadius: 'var(--radius-sm)',
        border: '1px solid var(--accent)',
        background: 'var(--accent-soft)',
        color: 'var(--accent)',
        fontSize: 12,
        fontWeight: 600,
        cursor: 'pointer',
        display: 'inline-flex',
        alignItems: 'center',
        gap: 6,
      }}
    >
      <FolderOpen size={14} weight="regular" aria-hidden="true" />
      Open folder
    </button>
  );
}
