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
    await page.waitForLoadState('networkidle');
  });

  test('lands directly on canvas without a login screen', async ({ page }) => {
    // Detect whether NO_AUTH is active by looking for the banner. If absent,
    // the backend/frontend isn't in no-auth mode — skip instead of failing.
    const banner = page.locator('.auth-banner').first();
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
    const banner = page.locator('.auth-banner').first();
    const bannerVisible = await banner
      .isVisible({ timeout: 2000 })
      .catch(() => false);
    test.skip(!bannerVisible, 'Frontend not running with VITE_NO_AUTH=true');

    await expect(banner).toBeVisible();

    // Spec: 32px full-width strip, #f87171 background.
    const box = await banner.boundingBox();
    expect(box?.height).toBe(32);

    const bg = await banner.evaluate((el) => getComputedStyle(el).backgroundColor);
    // rgb(248, 113, 113) = #f87171
    expect(bg).toBe('rgb(248, 113, 113)');
  });

  test('top bar is below the banner without overlap', async ({ page }) => {
    const banner = page.locator('.auth-banner').first();
    const bannerVisible = await banner.isVisible({ timeout: 2000 }).catch(() => false);
    test.skip(!bannerVisible, 'Frontend not running with VITE_NO_AUTH=true');

    const bannerBox = await banner.boundingBox();
    const header = page.locator('header').first();
    const headerBox = await header.boundingBox();

    const bannerBottom = (bannerBox?.y ?? 0) + (bannerBox?.height ?? 0);
    expect(headerBox?.y).toBeGreaterThanOrEqual(bannerBottom);
  });

  test('sidebar palette shows icon-grid (not text list)', async ({ page }) => {
    const banner = page.locator('.auth-banner').first();
    const bannerVisible = await banner.isVisible({ timeout: 2000 }).catch(() => false);
    test.skip(!bannerVisible, 'Frontend not running with VITE_NO_AUTH=true');

    // Palette aside should be visible.
    await expect(page.locator('.clotho-palette')).toBeVisible();

    // It should contain tile grids, not a bare list.
    const tileGrids = page.locator('.clotho-tile-grid');
    const count = await tileGrids.count();
    expect(count).toBeGreaterThanOrEqual(2); // at minimum Agent + Media grids

    // Media section tiles exist.
    const tiles = page.locator('.clotho-tile-label');
    const tileCount = await tiles.count();
    expect(tileCount).toBeGreaterThanOrEqual(3);
  });

  test('no blue or purple gradients in rendered CSS', async ({ page }) => {
    const banner = page.locator('.auth-banner').first();
    const bannerVisible = await banner.isVisible({ timeout: 2000 }).catch(() => false);
    test.skip(!bannerVisible, 'Frontend not running with VITE_NO_AUTH=true');

    const blueGradients = await page.evaluate(() => {
      const found: string[] = [];
      try {
        for (const sheet of Array.from(document.styleSheets)) {
          try {
            for (const rule of Array.from(sheet.cssRules ?? [])) {
              const text = rule.cssText;
              if (
                text.includes('gradient') &&
                (text.match(/#[0-9a-f]{6}/i)?.[0]?.match(/^#[2-9a-f][0-9a-f][4-9a-f]/i) ||
                  text.toLowerCase().includes('blue') ||
                  text.toLowerCase().includes('purple') ||
                  text.toLowerCase().includes('violet') ||
                  text.includes('#6366') ||
                  text.includes('#3b82'))
              ) {
                found.push(text.substring(0, 120));
              }
            }
          } catch {
            // cross-origin sheet — skip
          }
        }
      } catch {
        // ignore
      }
      return found;
    });

    expect(blueGradients).toHaveLength(0);
  });
});
