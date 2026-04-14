import { useState } from 'react';
import './AuthBanner.css';

const DISMISS_KEY = 'clotho.unauth-banner.dismissed';

function isDismissed(): boolean {
  if (typeof window === 'undefined') return false;
  try {
    return window.sessionStorage.getItem(DISMISS_KEY) === '1';
  } catch {
    return false;
  }
}

/**
 * AuthBanner — renders only when VITE_NO_AUTH === 'true'. A 32px strip pinned
 * at the top of the viewport that reminds the operator the app is in
 * unauthenticated mode.
 *
 * Dismissal is session-only: sessionStorage['clotho.unauth-banner.dismissed']
 * is cleared on tab close / reload, restoring the banner. The UnauthChip in
 * the top bar is the permanent signal that persists across dismissal.
 */
export function AuthBanner() {
  const [dismissed, setDismissed] = useState<boolean>(() => isDismissed());

  if (import.meta.env.VITE_NO_AUTH !== 'true') {
    return null;
  }

  if (dismissed) {
    return null;
  }

  const handleDismiss = () => {
    try {
      window.sessionStorage.setItem(DISMISS_KEY, '1');
    } catch {
      // Swallow: sessionStorage may be unavailable (private mode, SSR).
    }
    setDismissed(true);
  };

  return (
    <div
      className="auth-banner"
      role="status"
      aria-live="polite"
    >
      <span className="auth-banner__message">
        UNAUTHENTICATED MODE — DO NOT USE WITH REAL DATA
      </span>
      <button
        type="button"
        className="auth-banner__close"
        aria-label="Dismiss unauthenticated mode banner"
        onClick={handleDismiss}
      >
        {'\u00D7'}
      </button>
    </div>
  );
}
