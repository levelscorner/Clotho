import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup } from '@testing-library/react';
import { UnauthChip } from '../UnauthChip';

describe('UnauthChip', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    cleanup();
  });

  describe('when VITE_NO_AUTH is "true"', () => {
    beforeEach(() => {
      vi.stubEnv('VITE_NO_AUTH', 'true');
    });

    it('renders the UNAUTH pill with accessible label', () => {
      const { getByRole } = render(<UnauthChip />);
      const chip = getByRole('status', {
        name: /unauthenticated mode indicator/i,
      });
      expect(chip).toBeInTheDocument();
      expect(chip).toHaveTextContent('UNAUTH');
    });

    it('exposes a native title tooltip explaining the mode', () => {
      const { getByRole } = render(<UnauthChip />);
      const chip = getByRole('status');
      expect(chip.getAttribute('title')).toMatch(/unauthenticated dev mode/i);
      expect(chip.getAttribute('title')).toMatch(/NO_AUTH=false/);
    });

    it('has the unauth-chip class for token-driven styling', () => {
      const { getByRole } = render(<UnauthChip />);
      expect(getByRole('status')).toHaveClass('unauth-chip');
    });
  });

  describe('when VITE_NO_AUTH is not "true"', () => {
    it('renders nothing when VITE_NO_AUTH is "false"', () => {
      vi.stubEnv('VITE_NO_AUTH', 'false');
      const { container } = render(<UnauthChip />);
      expect(container.firstChild).toBeNull();
    });

    it('renders nothing when VITE_NO_AUTH is unset', () => {
      vi.stubEnv('VITE_NO_AUTH', '');
      const { container } = render(<UnauthChip />);
      expect(container.firstChild).toBeNull();
    });
  });
});
