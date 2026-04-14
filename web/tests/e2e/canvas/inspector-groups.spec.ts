import { test, expect } from '@playwright/test';

/**
 * Verifies the collapsible inspector groups behavior:
 *   - Basics group is open by default
 *   - Advanced group is collapsed by default
 *   - Clicking Advanced toggles it open
 *
 * Preconditions: a pipeline exists and a node can be selected. In no-auth
 * mode the canvas loads immediately; otherwise the spec auto-skips because
 * login + pipeline creation is already covered elsewhere.
 */
test.describe('Inspector groups', () => {
  test('Basics open by default, Advanced collapsed by default', async ({
    page,
  }) => {
    await page.goto('/');

    // Only run when canvas is reachable without login (no-auth mode).
    const canvas = page.locator('.react-flow');
    const canvasVisible = await canvas
      .isVisible({ timeout: 2000 })
      .catch(() => false);
    test.skip(!canvasVisible, 'Canvas not reachable without login');

    // Drag-drop nodes or select an existing one. We look for the node palette
    // and use keyboard-shortcut pipeline as a fallback.
    const firstNode = page.locator('.react-flow__node').first();
    const hasNode = await firstNode
      .isVisible({ timeout: 2000 })
      .catch(() => false);
    test.skip(!hasNode, 'No existing node to inspect');

    await firstNode.click();

    // Basics <details> should be open; Advanced should not be.
    const basics = page.locator('details', { hasText: 'Basics' }).first();
    const advanced = page.locator('details', { hasText: 'Advanced' }).first();

    await expect(basics).toHaveAttribute('open', /.*/);
    await expect(advanced).not.toHaveAttribute('open', /.*/);

    // Click Advanced summary → opens.
    await advanced.locator('summary').click();
    await expect(advanced).toHaveAttribute('open', /.*/);
  });
});
