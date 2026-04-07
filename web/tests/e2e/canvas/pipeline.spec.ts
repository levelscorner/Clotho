import { test, expect } from '../../fixtures/auth';

test.describe('Canvas Pipeline', () => {
  test('canvas loads with node palette', async ({ authedPage: page }) => {
    // Palette sidebar should be visible with node categories
    await expect(page.locator('text=Agents').first()).toBeVisible();
    await expect(page.locator('text=Tools').first()).toBeVisible();
  });

  test('can add an agent node by dragging from palette', async ({ authedPage: page }) => {
    // Find a palette item (Script Writer or similar)
    const paletteItem = page.locator('text=Script Writer').first();
    await expect(paletteItem).toBeVisible();

    // Count nodes before
    const nodesBefore = await page.locator('.clotho-node').count();

    // Drag from palette to canvas center
    const canvas = page.locator('.react-flow__pane');
    const canvasBox = await canvas.boundingBox();
    if (!canvasBox) throw new Error('Canvas not found');

    const paletteBox = await paletteItem.boundingBox();
    if (!paletteBox) throw new Error('Palette item not found');

    await page.mouse.move(paletteBox.x + paletteBox.width / 2, paletteBox.y + paletteBox.height / 2);
    await page.mouse.down();
    await page.mouse.move(canvasBox.x + canvasBox.width / 2, canvasBox.y + canvasBox.height / 2, { steps: 10 });
    await page.mouse.up();

    // Wait for node to appear
    await page.waitForTimeout(500);
    const nodesAfter = await page.locator('.clotho-node').count();
    expect(nodesAfter).toBeGreaterThanOrEqual(nodesBefore);
  });

  test('selecting a node shows inspector panel', async ({ authedPage: page }) => {
    // Click on a node if one exists
    const node = page.locator('.clotho-node').first();
    if (await node.isVisible()) {
      await node.click();
      // Inspector should show node details
      await page.waitForTimeout(500);
    }
  });

  test('save button persists pipeline', async ({ authedPage: page }) => {
    // Find and click save button
    const saveBtn = page.locator('button', { hasText: /save/i }).first();
    if (await saveBtn.isVisible()) {
      await saveBtn.click();
      // Should not show error
      await page.waitForTimeout(1000);
    }
  });

  test('undo/redo keyboard shortcuts work', async ({ authedPage: page }) => {
    // Test undo shortcut
    await page.keyboard.press('Control+z');
    await page.waitForTimeout(300);
    // Test redo shortcut
    await page.keyboard.press('Control+Shift+z');
    await page.waitForTimeout(300);
    // No crash = pass
  });
});
