import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/react';
import { AuthBanner } from '../AuthBanner';

const DISMISS_KEY = 'clotho.unauth-banner.dismissed';

describe('AuthBanner', () => {
  beforeEach(() => {
    vi.stubEnv('VITE_NO_AUTH', 'true');
    window.sessionStorage.clear();
  });

  afterEach(() => {
    vi.unstubAllEnvs();
    window.sessionStorage.clear();
    cleanup();
  });

  it('renders when VITE_NO_AUTH is "true" and not yet dismissed', () => {
    const { getByRole } = render(<AuthBanner />);
    const banner = getByRole('status');
    expect(banner).toHaveTextContent(/UNAUTHENTICATED MODE/);
  });

  it('renders the close button with the proper aria-label', () => {
    const { getByRole } = render(<AuthBanner />);
    const closeBtn = getByRole('button', {
      name: /dismiss unauthenticated mode banner/i,
    });
    expect(closeBtn).toBeInTheDocument();
  });

  it('hides the banner and writes to sessionStorage when the close button is clicked', () => {
    const { getByRole, queryByRole } = render(<AuthBanner />);
    const closeBtn = getByRole('button', {
      name: /dismiss unauthenticated mode banner/i,
    });

    fireEvent.click(closeBtn);

    expect(queryByRole('status')).toBeNull();
    expect(window.sessionStorage.getItem(DISMISS_KEY)).toBe('1');
  });

  it('does not render when sessionStorage already has the dismiss flag on mount', () => {
    window.sessionStorage.setItem(DISMISS_KEY, '1');
    const { container } = render(<AuthBanner />);
    expect(container.firstChild).toBeNull();
  });

  it('renders null when VITE_NO_AUTH is not "true"', () => {
    vi.stubEnv('VITE_NO_AUTH', 'false');
    const { container } = render(<AuthBanner />);
    expect(container.firstChild).toBeNull();
  });
});
