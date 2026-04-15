import { test, expect } from '@playwright/test';

/**
 * Two bug-fixes verified here:
 *
 * 1. Drag-drop from the palette lands under the cursor (via React Flow's
 *    screenToFlowPosition), not at the canvas origin. Previous behaviour
 *    subtracted the canvas bounds from clientX/Y without accounting for
 *    pan/zoom.
 *
 * 2. Right-click anywhere on a node opens the actions menu at the cursor
 *    position. Previous behaviour anchored the menu to the three-dot
 *    button regardless of where the click happened.
 */

test.describe('Drop + right-click fixes', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    // Dismiss auth banner so clicks/drags are not intercepted.
    const close = page.locator('.auth-banner__close');
    if (await close.isVisible({ timeout: 1000 }).catch(() => false)) {
      await close.click();
    }
    // Load sample pipeline so we have nodes for the right-click test.
    const cta = page.locator('.empty-canvas__cta');
    if (await cta.isVisible({ timeout: 2000 }).catch(() => false)) {
      await cta.click();
      await page.waitForTimeout(600);
    }
  });

  test('dropped node lands near cursor, not at canvas origin', async ({ page }) => {
    // Open the tools flyout so the Text tile is available.
    await page.locator('[data-testid="rail-tools"]').click();
    await page.waitForTimeout(200);

    const textTile = page.locator('.clotho-tile-grid')
      .locator('div')
      .filter({ hasText: /^Text$/ })
      .first();
    await expect(textTile).toBeVisible();

    const before = await page.locator('.react-flow__node').count();

    // Aim for a point roughly centered in the visible canvas area. The
    // actual x/y doesn't matter for the assertion; we just need a
    // reproducible target and then we check the node ends up within
    // a reasonable radius of it.
    const canvas = page.locator('.react-flow__pane').first();
    const canvasBox = await canvas.boundingBox();
    if (!canvasBox) test.skip(true, 'canvas not laid out');

    const targetX = canvasBox!.x + canvasBox!.width * 0.7;
    const targetY = canvasBox!.y + canvasBox!.height * 0.3;

    const tileBox = await textTile.boundingBox();
    if (!tileBox) test.skip(true, 'tile not laid out');

    // Simulate a drag from the tile to (targetX, targetY).
    await page.mouse.move(tileBox!.x + 20, tileBox!.y + 20);
    await page.mouse.down();
    await page.mouse.move(targetX, targetY, { steps: 10 });
    await page.mouse.up();
    await page.waitForTimeout(500);

    const after = await page.locator('.react-flow__node').count();
    expect(after).toBe(before + 1);

    // React Flow positions nodes by their top-left corner. After drop,
    // the new node's top-left in screen space should equal the cursor
    // location that was passed through screenToFlowPosition and back —
    // tolerance of 50px absorbs any minor subpixel/zoom rounding.
    const newNode = page.locator('.react-flow__node').last();
    const nodeBox = await newNode.boundingBox();
    expect(nodeBox).not.toBeNull();

    const dxTopLeft = Math.abs(nodeBox!.x - targetX);
    const dyTopLeft = Math.abs(nodeBox!.y - targetY);
    expect(dxTopLeft).toBeLessThan(50);
    expect(dyTopLeft).toBeLessThan(50);
  });

  test('right-click on node body opens menu near cursor, not at button', async ({ page }) => {
    const node = page.locator('.react-flow__node').first();
    await expect(node).toBeVisible({ timeout: 5000 });

    const nodeBox = await node.boundingBox();
    if (!nodeBox) test.skip(true, 'node not laid out');

    // Right-click deep inside the node body — far from the three-dot
    // button (which sits top-right). Aim for the bottom-left quadrant.
    const clickX = nodeBox!.x + nodeBox!.width * 0.25;
    const clickY = nodeBox!.y + nodeBox!.height * 0.75;

    await page.mouse.click(clickX, clickY, { button: 'right' });
    await page.waitForTimeout(150);

    const menu = page.locator('.clotho-menu').first();
    await expect(menu).toBeVisible({ timeout: 2000 });

    // The menu should render near the click coordinates, not up in the
    // top-right corner where the three-dot button lives. We allow a
    // generous 300px radius to keep the test stable across browsers
    // and zoom levels.
    const menuBox = await menu.boundingBox();
    expect(menuBox).not.toBeNull();

    const menuCentreX = menuBox!.x + menuBox!.width / 2;
    const menuCentreY = menuBox!.y + menuBox!.height / 2;
    const dx = Math.abs(menuCentreX - clickX);
    const dy = Math.abs(menuCentreY - clickY);
    expect(dx).toBeLessThan(300);
    expect(dy).toBeLessThan(300);

    // The menu contents should match the three-dot variant so behaviour
    // stays in sync — Duplicate + Rename + Lock + Delete at minimum.
    const labels = await menu.locator('.clotho-menu__item').allTextContents();
    const joined = labels.join(' ');
    expect(joined).toMatch(/Duplicate/);
    expect(joined).toMatch(/Rename/);
    expect(joined).toMatch(/Lock/);
    expect(joined).toMatch(/Delete/);

    // Close.
    await page.keyboard.press('Escape');
    await expect(menu).not.toBeVisible({ timeout: 2000 });
  });
});
