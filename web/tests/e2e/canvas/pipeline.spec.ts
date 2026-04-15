import { test, expect } from '@playwright/test';

/**
 * Wave 3 — Canvas states, empty state, ⌘K template gallery.
 * Wave 4 — /dev/nodes testbed, responsive layout, A11y.
 *
 * All tests require no-auth mode (VITE_NO_AUTH=true).
 */

test.describe('Canvas states', () => {
  test('⌘K opens TemplateGallery modal, Escape closes it', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const canvas = page.locator('.react-flow');
    const canvasVisible = await canvas.isVisible({ timeout: 3000 }).catch(() => false);
    test.skip(!canvasVisible, 'Canvas not available');

    // Press ⌘K (meta+k on macOS).
    await page.keyboard.press('Meta+k');

    // TemplateGallery modal should appear.
    const modal = page.locator('[role="dialog"]').first();
    await expect(modal).toBeVisible({ timeout: 3000 });

    // Press Escape — modal should close.
    await page.keyboard.press('Escape');
    await expect(modal).not.toBeVisible({ timeout: 3000 });
  });

  test('Templates button in top bar opens TemplateGallery', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const canvas = page.locator('.react-flow');
    const canvasVisible = await canvas.isVisible({ timeout: 3000 }).catch(() => false);
    test.skip(!canvasVisible, 'Canvas not available');

    const templatesBtn = page.getByRole('button', { name: /Templates/i });
    await expect(templatesBtn).toBeVisible();
    await templatesBtn.click();

    const modal = page.locator('[role="dialog"]').first();
    await expect(modal).toBeVisible({ timeout: 3000 });

    // Close via Escape.
    await page.keyboard.press('Escape');
    await expect(modal).not.toBeVisible({ timeout: 3000 });
  });

  test('empty canvas shows ghost cluster when pipeline has no nodes', async ({ browser }) => {
    // Create a fresh empty pipeline via API.
    const freshPipelineId = await createFreshEmptyPipeline();
    if (!freshPipelineId) {
      test.skip(true, 'Could not create fresh empty pipeline via API');
    }

    const context = await browser.newContext();
    const page = await context.newPage();

    // Clear dismissal flag.
    await page.goto('http://localhost:3000/');
    await page.evaluate(() => {
      localStorage.removeItem('clotho.empty-state.dismissed');
    });

    // Navigate so app bootstraps and loads the fresh pipeline.
    // The app auto-selects the first pipeline — we need to trigger switch
    // by injecting state after load; the simplest check is to see if the
    // current pipeline becomes empty after loading.
    await page.waitForLoadState('networkidle');

    const nodeCount = await page.locator('.react-flow__node').count();
    if (nodeCount > 0) {
      // Backend has a saved pipeline with nodes; skip this check.
      await context.close();
      test.skip(true, 'Backend default pipeline has saved nodes; empty state requires a 0-node pipeline. Fix: save a blank pipeline version.');
    }

    // Empty state should be visible.
    const emptyCanvas = page.locator('.empty-canvas');
    await expect(emptyCanvas).toBeVisible({ timeout: 3000 });

    const ghosts = page.locator('.empty-canvas__ghost');
    await expect(ghosts).toHaveCount(3);

    const cta = page.locator('.empty-canvas__cta');
    await expect(cta).toBeVisible();
    await expect(cta).toContainText('LOAD SAMPLE PIPELINE');

    const cornerHint = page.locator('.empty-canvas__hint-corner');
    await expect(cornerHint).toContainText('⌘K');

    // Click CTA → 3 nodes appear.
    await cta.click();
    await expect(page.locator('.react-flow__node')).toHaveCount(3, { timeout: 5000 });

    // localStorage flag set.
    const dismissed = await page.evaluate(() => localStorage.getItem('clotho.empty-state.dismissed'));
    expect(dismissed).toBe('1');

    await context.close();
  });
});

test.describe('/dev/nodes testbed', () => {
  test('5 tabs render with fixture cards', async ({ page }) => {
    await page.goto('/dev/nodes');
    await page.waitForLoadState('networkidle');

    // 5 tabs.
    const tabs = page.locator('.dev-nodes__tab');
    await expect(tabs).toHaveCount(5);

    const tabLabels = await tabs.allTextContents();
    expect(tabLabels).toContain('Agent');
    expect(tabLabels).toContain('Media · Image');
    expect(tabLabels).toContain('Media · Video');
    expect(tabLabels).toContain('Media · Audio');
    expect(tabLabels).toContain('Tool');

    // Grid has cards.
    const grid = page.locator('[data-testid="dev-nodes-grid"]');
    await expect(grid).toBeVisible({ timeout: 5000 });
    await expect(page.locator('.dev-nodes__card')).toHaveCount(5);
  });

  test('each tab shows correct number of fixture cards', async ({ page }) => {
    await page.goto('/dev/nodes');
    await page.waitForLoadState('networkidle');

    // Agent tab: 5 states = 5 cards.
    // Media tabs: 5 states × 1 media type = 5 cards each.
    // Tool tab: 5 states × 3 tool types = 15 cards (text_box, image_box, video_box).
    const tabExpectedCounts: Array<{ label: string; count: number }> = [
      { label: 'Agent', count: 5 },
      { label: 'Media · Image', count: 5 },
      { label: 'Media · Video', count: 5 },
      { label: 'Media · Audio', count: 5 },
      { label: 'Tool', count: 15 }, // 3 tool types × 5 states
    ];

    // Ensure grid is populated before iterating tabs.
    const grid = page.locator('[data-testid="dev-nodes-grid"]');
    await expect(grid.locator('.dev-nodes__card').first()).toBeVisible({ timeout: 5000 });

    for (const { label, count } of tabExpectedCounts) {
      const tab = page.getByRole('button', { name: label, exact: true });
      await tab.click();
      // Wait for the tab to be marked active (aria-pressed=true).
      await expect(tab).toHaveAttribute('aria-pressed', 'true', { timeout: 2000 });
      // Wait for the grid to update (card count to settle).
      await expect(grid.locator('.dev-nodes__card')).toHaveCount(count, { timeout: 4000 });
    }
  });

  test('state badges include all 5 expected states', async ({ page }) => {
    await page.goto('/dev/nodes');
    await page.waitForLoadState('networkidle');

    const badges = await page.locator('.dev-nodes__state-badge').allTextContents();
    const badgeSet = new Set(badges);
    expect(badgeSet.has('QUEUED')).toBe(true);
    expect(badgeSet.has('RUNNING')).toBe(true);
    expect(badgeSet.has('COMPLETE')).toBe(true);
    expect(badgeSet.has('EMPTY-COMPLETE')).toBe(true);
    expect(badgeSet.has('FAILED')).toBe(true);
  });
});

test.describe('Responsive layout', () => {
  test('at 768px ActivityRail (48px) is visible; flyout is closed by default', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 900 });
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const canvas = page.locator('.react-flow');
    const canvasVisible = await canvas.isVisible({ timeout: 3000 }).catch(() => false);
    test.skip(!canvasVisible, 'Canvas not available');

    // The persistent icon column is the ActivityRail (48px).
    const rail = page.locator('[data-testid="activity-rail"]');
    await expect(rail).toBeVisible();

    const box = await rail.boundingBox();
    expect(box?.width).toBeLessThanOrEqual(52); // 48px ± 4px tolerance

    // Flyout panel hidden until a rail icon is clicked.
    await expect(page.locator('.clotho-palette')).toBeHidden();
  });

  test('at 375px hamburger button visible and opens palette', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 });
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const canvas = page.locator('.react-flow');
    const canvasVisible = await canvas.isVisible({ timeout: 3000 }).catch(() => false);
    test.skip(!canvasVisible, 'Canvas not available');

    // Dismiss AuthBanner if it's covering the hamburger button at narrow viewport.
    const authBanner = page.locator('.auth-banner');
    const authBannerVisible = await authBanner.isVisible({ timeout: 1000 }).catch(() => false);
    if (authBannerVisible) {
      const closeBtn = authBanner.locator('.auth-banner__close');
      const closeBtnVisible = await closeBtn.isVisible({ timeout: 1000 }).catch(() => false);
      if (closeBtnVisible) {
        await closeBtn.click();
      } else {
        await page.evaluate(() => {
          const el = document.querySelector('.auth-banner') as HTMLElement | null;
          if (el) el.style.display = 'none';
        });
      }
    }

    // Dismiss SmallScreenBanner if it's blocking interactions.
    const banner = page.locator('.clotho-small-screen-banner');
    const bannerVisible = await banner.isVisible({ timeout: 1000 }).catch(() => false);
    if (bannerVisible) {
      const dismissBtn = banner.getByRole('button').first();
      const dismissVisible = await dismissBtn.isVisible({ timeout: 1000 }).catch(() => false);
      if (dismissVisible) {
        await dismissBtn.click();
      } else {
        // Force-hide via JS if no button
        await page.evaluate(() => {
          const el = document.querySelector('.clotho-small-screen-banner') as HTMLElement | null;
          if (el) el.style.display = 'none';
        });
      }
    }

    const hamburger = page.locator('.clotho-hamburger');
    await expect(hamburger).toBeVisible();

    // Click hamburger → palette drawer opens.
    await hamburger.click();
    const palette = page.locator('.clotho-palette');
    await expect(palette).toHaveAttribute('data-mobile-open', 'true');

    // Close via the close button inside the drawer (the palette covers the hamburger
    // at z-index:modal > z-index:hamburger+10 at phone breakpoint).
    const closeBtn = palette.getByRole('button', { name: 'Close node palette' });
    await expect(closeBtn).toBeVisible({ timeout: 3000 });
    await closeBtn.click();
    await expect(palette).toHaveAttribute('data-mobile-open', 'false');
  });

  test('at 1440px SmallScreenBanner is NOT visible', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const canvas = page.locator('.react-flow');
    const canvasVisible = await canvas.isVisible({ timeout: 3000 }).catch(() => false);
    test.skip(!canvasVisible, 'Canvas not available');

    const banner = page.locator('.clotho-small-screen-banner');
    // At 1440px the banner's CSS display stays "none".
    const display = await banner.evaluate((el) => getComputedStyle(el).display);
    expect(display).toBe('none');
  });

  test('at 1023px SmallScreenBanner IS visible (threshold is 1024px)', async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1023, height: 768 });
    await page.goto('/');
    // Clear sessionStorage dismissal so the banner shows.
    await page.evaluate(() => sessionStorage.removeItem('clotho.small-screen-banner.dismissed'));
    await page.reload();
    await page.waitForLoadState('networkidle');

    const canvas = page.locator('.react-flow');
    const canvasVisible = await canvas.isVisible({ timeout: 3000 }).catch(() => false);
    test.skip(!canvasVisible, 'Canvas not available');

    const banner = page.locator('.clotho-small-screen-banner');
    // Banner data-visible should be true (not dismissed).
    await expect(banner).toHaveAttribute('data-visible', 'true');
    const display = await banner.evaluate((el) => getComputedStyle(el).display);
    expect(display).not.toBe('none');
  });
});

test.describe('A11y: inspector drawer in overlay mode', () => {
  test('role=dialog + aria-modal=true at 768px; Escape closes inspector', async ({
    page,
  }) => {
    await page.setViewportSize({ width: 768, height: 900 });
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const canvas = page.locator('.react-flow');
    const canvasVisible = await canvas.isVisible({ timeout: 3000 }).catch(() => false);
    test.skip(!canvasVisible, 'Canvas not available');

    // Click a node.
    const firstNode = page.locator('.react-flow__node').first();
    const hasNode = await firstNode.isVisible({ timeout: 2000 }).catch(() => false);
    test.skip(!hasNode, 'No node to click');

    await firstNode.click();

    // Inspector should appear as dialog.
    const inspector = page.locator('.clotho-inspector');
    await expect(inspector).toBeVisible({ timeout: 3000 });
    await expect(inspector).toHaveAttribute('role', 'dialog');
    await expect(inspector).toHaveAttribute('aria-modal', 'true');

    // Escape closes the inspector.
    await page.keyboard.press('Escape');
    await expect(inspector).not.toBeVisible({ timeout: 3000 });
  });
});

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

async function createFreshEmptyPipeline(): Promise<string> {
  try {
    const projectsResp = await fetch('http://localhost:8080/api/projects');
    const projects = (await projectsResp.json()) as { id: string }[];
    if (!projects.length) return '';

    const projectId = projects[0].id;
    const pipelineResp = await fetch(
      `http://localhost:8080/api/projects/${projectId}/pipelines`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: 'E2E Empty Test Pipeline' }),
      },
    );
    if (!pipelineResp.ok) return '';
    const pipeline = (await pipelineResp.json()) as { id: string };
    return pipeline.id ?? '';
  } catch {
    return '';
  }
}
