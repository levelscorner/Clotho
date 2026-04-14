import { test, expect } from '@playwright/test';

/**
 * Visual regression sweep of /dev/nodes.
 *
 * SKIPPED in CI by default: screenshot-based tests are flaky unless the
 * baseline snapshots are committed and the render environment is pinned.
 * To (re)generate baselines locally, run:
 *   PLAYWRIGHT_UPDATE_SNAPSHOTS=1 npx playwright test visual-regression.spec.ts --update-snapshots
 * To run against committed baselines:
 *   RUN_VISUAL_REGRESSION=1 npx playwright test visual-regression.spec.ts
 */
const SHOULD_RUN =
  process.env.RUN_VISUAL_REGRESSION === '1' ||
  process.env.PLAYWRIGHT_UPDATE_SNAPSHOTS === '1';

test.describe('Visual regression — /dev/nodes', () => {
  test.skip(
    !SHOULD_RUN,
    'Visual regression skipped — set RUN_VISUAL_REGRESSION=1 or PLAYWRIGHT_UPDATE_SNAPSHOTS=1',
  );

  const tabs = ['agents', 'media', 'tools'] as const;

  for (const tab of tabs) {
    test(`matches baseline snapshot for tab=${tab}`, async ({ page }) => {
      await page.goto(`/dev/nodes?tab=${tab}`);

      // Wait for at least one node card to render.
      await expect(page.locator('[data-testid="dev-nodes-grid"]')).toBeVisible({
        timeout: 5000,
      });

      // Give animations a beat to settle.
      await page.waitForTimeout(300);

      await expect(page).toHaveScreenshot(`dev-nodes-${tab}.png`, {
        fullPage: true,
        maxDiffPixelRatio: 0.01,
      });
    });
  }
});
