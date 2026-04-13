import './AuthBanner.css';

/**
 * AuthBanner — renders only when VITE_NO_AUTH === 'true'. A 32px strip pinned
 * at the top of the viewport that reminds the operator the app is in
 * unauthenticated mode. Non-dismissible on purpose: the banner IS the safety.
 *
 * Layout strategy: fixed positioning at the top of the viewport. The app root
 * adds 32px of top padding via the `.app-root--no-auth` modifier so the top
 * bar is not occluded. See App.tsx.
 */
export function AuthBanner() {
  if (import.meta.env.VITE_NO_AUTH !== 'true') {
    return null;
  }

  return (
    <div
      className="auth-banner"
      role="status"
      aria-live="polite"
    >
      UNAUTHENTICATED MODE — DO NOT USE WITH REAL DATA
    </div>
  );
}
