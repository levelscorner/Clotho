import { useEffect, useRef } from 'react';

// Selector of all elements we consider tabbable inside a dialog. Mirrors
// what screen-reader users and keyboard users hit with Tab.
const TABBABLE =
  'a[href], button:not([disabled]), textarea:not([disabled]), ' +
  'input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

/**
 * Trap keyboard focus inside the given container while `open` is true.
 *
 * On open:
 *   - Remembers the element that previously had focus.
 *   - Moves focus to the first tabbable descendant (or the container if
 *     nothing tabbable exists — user can still Escape out).
 *
 * While open:
 *   - Tab cycles through tabbable descendants only; never escapes into
 *     the rest of the page.
 *
 * On close:
 *   - Restores focus to the previously-focused element.
 *
 * This is the minimum useful focus-trap behaviour — not a fully-featured
 * library. It covers our TemplateGallery + SettingsPanel modals which
 * render a handful of interactive elements each.
 */
export function useFocusTrap(
  containerRef: React.RefObject<HTMLElement>,
  open: boolean,
): void {
  const previousActive = useRef<Element | null>(null);

  useEffect(() => {
    if (!open) return;
    const container = containerRef.current;
    if (!container) return;

    previousActive.current = document.activeElement;

    // Focus the first tabbable child, or the container itself.
    const first = container.querySelector<HTMLElement>(TABBABLE);
    if (first) {
      first.focus();
    } else {
      // Make the container programmatically focusable so focus doesn't
      // fall outside the modal entirely.
      if (!container.hasAttribute('tabindex')) {
        container.setAttribute('tabindex', '-1');
      }
      container.focus();
    }

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Tab') return;
      const tabbables = Array.from(
        container.querySelectorAll<HTMLElement>(TABBABLE),
      ).filter((el) => !el.hasAttribute('disabled'));
      if (tabbables.length === 0) {
        e.preventDefault();
        return;
      }
      const firstEl = tabbables[0];
      const lastEl = tabbables[tabbables.length - 1];
      const active = document.activeElement as HTMLElement | null;
      if (e.shiftKey && active === firstEl) {
        e.preventDefault();
        lastEl.focus();
      } else if (!e.shiftKey && active === lastEl) {
        e.preventDefault();
        firstEl.focus();
      }
    };

    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('keydown', onKeyDown);
      const prev = previousActive.current;
      if (prev && prev instanceof HTMLElement) {
        prev.focus();
      }
    };
  }, [open, containerRef]);
}
