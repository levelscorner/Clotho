import { useEffect } from 'react';
import { useUIStore } from '../stores/uiStore';
import { usePipelineStore } from '../stores/pipelineStore';

// ---------------------------------------------------------------------------
// Global keyboard shortcuts
//
// Attach once near the root of the authenticated app. Current bindings:
//   - ⌘K / Ctrl+K   toggle the template gallery
//   - ⌘D / Ctrl+D   duplicate the selected node
//   - F2            rename the selected node (opens inline rename state)
//   - ⌘L / Ctrl+L   toggle lock on the selected node
//   - Escape        close the template gallery if it's open, or cancel rename
//
// All shortcuts are skipped while the user is typing in an input, textarea,
// or contenteditable element so they don't hijack keystrokes mid-edit.
// ---------------------------------------------------------------------------

function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  const tag = target.tagName.toLowerCase();
  if (tag === 'input' || tag === 'textarea' || tag === 'select') return true;
  if (target.isContentEditable) return true;
  return false;
}

// Exported for tests — the handler is pure over the event + stores.
export function handleGlobalKeydown(e: KeyboardEvent): void {
  const typing = isTypingTarget(e.target);
  const mod = e.metaKey || e.ctrlKey;
  const key = e.key.toLowerCase();

  // ⌘K on macOS, Ctrl+K elsewhere.
  if (mod && key === 'k') {
    if (typing) return;
    e.preventDefault();
    useUIStore.getState().toggleTemplateGallery();
    return;
  }

  // ⌘D — duplicate selected node
  if (mod && key === 'd') {
    if (typing) return;
    const { selectedNodeId, duplicateNode } = usePipelineStore.getState();
    if (!selectedNodeId) return;
    e.preventDefault();
    duplicateNode(selectedNodeId);
    return;
  }

  // ⌘L — toggle lock on selected node
  if (mod && key === 'l') {
    if (typing) return;
    const { selectedNodeId, toggleLock } = usePipelineStore.getState();
    if (!selectedNodeId) return;
    e.preventDefault();
    toggleLock(selectedNodeId);
    return;
  }

  // F2 — start rename on selected node
  if (e.key === 'F2') {
    if (typing) return;
    const { selectedNodeId, startRename } = usePipelineStore.getState();
    if (!selectedNodeId) return;
    e.preventDefault();
    startRename(selectedNodeId);
    return;
  }

  if (e.key === 'Escape') {
    const uiState = useUIStore.getState();
    if (uiState.templateGalleryOpen) {
      e.preventDefault();
      uiState.closeTemplateGallery();
      return;
    }
    const pipelineState = usePipelineStore.getState();
    if (pipelineState.renamingNodeId) {
      e.preventDefault();
      pipelineState.cancelRename();
    }
  }
}

export function useGlobalKeyboardShortcuts(): void {
  useEffect(() => {
    document.addEventListener('keydown', handleGlobalKeydown);
    return () => {
      document.removeEventListener('keydown', handleGlobalKeydown);
    };
  }, []);
}
