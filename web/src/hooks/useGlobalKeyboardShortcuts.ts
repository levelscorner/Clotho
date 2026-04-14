import { useEffect } from 'react';
import { useUIStore } from '../stores/uiStore';

// ---------------------------------------------------------------------------
// Global keyboard shortcuts
//
// Attach once near the root of the authenticated app. Current bindings:
//   - ⌘K / Ctrl+K   toggle the template gallery (skipped while typing in an
//                   input, textarea, or contenteditable element)
//   - Escape        close the template gallery if it's open (other Escape
//                   handlers, e.g. TemplateGallery's own, continue to run)
// ---------------------------------------------------------------------------

function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  const tag = target.tagName.toLowerCase();
  if (tag === 'input' || tag === 'textarea' || tag === 'select') return true;
  if (target.isContentEditable) return true;
  return false;
}

// Exported for tests — the handler is pure over the event + store.
export function handleGlobalKeydown(e: KeyboardEvent): void {
  // ⌘K on macOS, Ctrl+K elsewhere.
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    if (isTypingTarget(e.target)) return;
    e.preventDefault();
    useUIStore.getState().toggleTemplateGallery();
    return;
  }

  if (e.key === 'Escape') {
    const state = useUIStore.getState();
    if (state.templateGalleryOpen) {
      e.preventDefault();
      state.closeTemplateGallery();
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
