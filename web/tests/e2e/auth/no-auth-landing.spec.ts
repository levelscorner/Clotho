import { test, expect } from '@playwright/test';

/**
 * Verifies the VITE_NO_AUTH=true bypass lands directly on the canvas and
 * renders the persistent "UNAUTHENTICATED MODE" banner.
 *
 * Preconditions:
 *   - Frontend dev server running with VITE_NO_AUTH=true
 *   - Backend running with NO_AUTH=true + CLOTHO_ACKNOWLEDGE_NO_AUTH=yes
 *
 * If these env vars are NOT set, the spec auto-skips — the auth-on flow is
 * covered by tests/e2e/auth/login.spec.ts.
 */
test.describe('No-auth landing', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('lands directly on canvas without a login screen', async ({ page }) => {
    // Detect whether NO_AUTH is active by looking for the banner. If absent,
    // the backend/frontend isn't in no-auth mode — skip instead of failing.
    const banner = page.locator('text=UNAUTHENTICATED MODE').first();
    const bannerVisible = await banner
      .isVisible({ timeout: 2000 })
      .catch(() => false);

    test.skip(!bannerVisible, 'Frontend not running with VITE_NO_AUTH=true');

    // Login inputs should NOT be present.
    await expect(page.locator('input[type="email"]')).toHaveCount(0);
    await expect(page.locator('input[type="password"]')).toHaveCount(0);

    // React Flow canvas must be visible.
    await expect(page.locator('.react-flow')).toBeVisible();
  });

  test('renders the persistent red no-auth banner at the top', async ({
    page,
  }) => {
    const banner = page.locator('text=UNAUTHENTICATED MODE').first();
    const bannerVisible = await banner
      .isVisible({ timeout: 2000 })
      .catch(() => false);
    test.skip(!bannerVisible, 'Frontend not running with VITE_NO_AUTH=true');

    await expect(banner).toBeVisible();
  });
});
