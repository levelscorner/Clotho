import './UnauthChip.css';

const TOOLTIP =
  'Unauthenticated dev mode — no login required. Flip NO_AUTH=false to enable auth.';

/**
 * UnauthChip — a persistent top-bar pill that signals the app is running in
 * unauthenticated dev mode. Visible whenever VITE_NO_AUTH === 'true', even
 * after the session dismisses the AuthBanner. Reload/new tab restores the
 * banner; this chip never dismisses.
 */
export function UnauthChip() {
  if (import.meta.env.VITE_NO_AUTH !== 'true') {
    return null;
  }

  return (
    <span
      className="unauth-chip"
      role="status"
      aria-label="Unauthenticated mode indicator"
      title={TOOLTIP}
    >
      UNAUTH
    </span>
  );
}
