import { useEffect, useState } from 'react';
import { useUIStore } from '../../stores/uiStore';

const SMALL_SCREEN_KEY = 'clotho.small-screen-banner.dismissed';
const PHONE_HINT_KEY = 'clotho.phone-hint.dismissed';

/**
 * SmallScreenBanner — under 1024px, shows a dismissible informational banner.
 * Dismissal is per-session (sessionStorage) so a page reload during the same
 * session keeps the banner hidden, but a fresh tab shows it again.
 */
export function SmallScreenBanner() {
  const [dismissed, setDismissed] = useState<boolean>(() => {
    if (typeof window === 'undefined') return false;
    try {
      return window.sessionStorage.getItem(SMALL_SCREEN_KEY) === '1';
    } catch {
      return false;
    }
  });

  const dismiss = () => {
    setDismissed(true);
    try {
      window.sessionStorage.setItem(SMALL_SCREEN_KEY, '1');
    } catch {
      // storage unavailable — tolerate silently
    }
  };

  return (
    <div
      className="clotho-small-screen-banner"
      data-visible={dismissed ? 'false' : 'true'}
      role="status"
    >
      Clotho is optimized for larger screens.
      <button type="button" aria-label="Dismiss" onClick={dismiss}>
        {'\u2715'}
      </button>
    </div>
  );
}

/**
 * PhoneHint — one-time dismissible hint on phone-sized viewports. Persisted
 * to localStorage so we don't keep nagging across sessions.
 */
export function PhoneHint() {
  const [dismissed, setDismissed] = useState<boolean>(() => {
    if (typeof window === 'undefined') return true;
    try {
      return window.localStorage.getItem(PHONE_HINT_KEY) === '1';
    } catch {
      return true;
    }
  });

  const [isPhone, setIsPhone] = useState<boolean>(() =>
    typeof window !== 'undefined'
      ? window.matchMedia('(max-width: 767px)').matches
      : false,
  );

  useEffect(() => {
    if (typeof window === 'undefined') return;
    const mq = window.matchMedia('(max-width: 767px)');
    const onChange = (e: MediaQueryListEvent) => setIsPhone(e.matches);
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  }, []);

  if (dismissed || !isPhone) return null;

  const onDismiss = () => {
    setDismissed(true);
    try {
      window.localStorage.setItem(PHONE_HINT_KEY, '1');
    } catch {
      // storage unavailable — tolerate silently
    }
  };

  return (
    <div className="clotho-phone-hint" role="status">
      <span style={{ flex: 1 }}>
        Drag and pinch work. Best on larger screens.
      </span>
      <button type="button" onClick={onDismiss}>
        Got it
      </button>
    </div>
  );
}


/**
 * MobileHamburger — icon-button visible only at phone breakpoint.
 *
 * Positioned fixed at the top-left of the viewport (styles in responsive.css,
 * class `.clotho-hamburger`). Tapping toggles the NodePalette as a left-edge
 * drawer. A11y: labeled button, `aria-expanded` reflects drawer state,
 * `aria-controls` points at the palette's id.
 *
 * Visibility is purely CSS-driven — at >=768px the class has `display:none`,
 * so we can render it unconditionally without layout leakage on desktop.
 */
export function MobileHamburger() {
  const open = useUIStore((s) => s.mobilePaletteOpen);
  const toggle = useUIStore((s) => s.toggleMobilePalette);

  return (
    <button
      type="button"
      className="clotho-hamburger"
      aria-label={open ? 'Close node palette' : 'Open node palette'}
      aria-expanded={open}
      aria-controls="clotho-node-palette"
      onClick={toggle}
    >
      {open ? '\u2715' : '\u2630'}
    </button>
  );
}
