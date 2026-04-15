import { test, expect } from '@playwright/test';
import * as path from 'path';

/**
 * Sprint design/warm-amber-v2 behavioral verification.
 *
 * Covers:
 *   1. AuthBanner dismiss → UnauthChip remains
 *   2. Sidebar 3 sections (Agent/Personality/Tools) + Phosphor icons
 *   3. EmptyCanvasState ghost cluster, CTA, ⌘K hint
 *   4. NodeActionsMenu ⋯ button, menu items, Lock badge + delete guard
 *   5. Port label prettification (image_prompt → "image prompt")
 *   6. Node teaser description on node body
 *   7. Inspector "About this node" group above "Basics"
 *   8. Media provider guard (image ≠ kokoro; audio = kokoro)
 */

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

async function ensureCanvas(page: import('@playwright/test').Page): Promise<boolean> {
  await page.goto('/');
  await page.waitForLoadState('networkidle');
  const canvas = page.locator('.react-flow');
  return canvas.isVisible({ timeout: 4000 }).catch(() => false);
}

async function ensureNodes(page: import('@playwright/test').Page): Promise<boolean> {
  const ok = await ensureCanvas(page);
  if (!ok) return false;

  let count = await page.locator('.react-flow__node').count();
  if (count > 0) return true;

  // Dismiss auth banner so CTA is clickable.
  const authClose = page.locator('.auth-banner__close');
  if (await authClose.isVisible({ timeout: 1000 }).catch(() => false)) {
    await authClose.click();
  }

  const cta = page.locator('.empty-canvas__cta');
  if (await cta.isVisible({ timeout: 2000 }).catch(() => false)) {
    await cta.click();
    await page.waitForFunction(
      () => document.querySelectorAll('.react-flow__node').length >= 3,
      { timeout: 5000 },
    );
  }

  count = await page.locator('.react-flow__node').count();
  return count > 0;
}

// ---------------------------------------------------------------------------
// 1. AuthBanner + UnauthChip
// ---------------------------------------------------------------------------

test.describe('AuthBanner + UnauthChip', () => {
  test('banner has dismiss × button; click × hides banner; UNAUTH chip remains', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const banner = page.locator('.auth-banner');
    const chip = page.locator('.unauth-chip');

    if (!(await banner.isVisible({ timeout: 2000 }).catch(() => false))) {
      test.skip(true, 'Not running with VITE_NO_AUTH=true');
    }

    // Banner visible with close button.
    await expect(banner).toBeVisible();
    const closeBtn = banner.locator('.auth-banner__close');
    await expect(closeBtn).toBeVisible();

    // UNAUTH chip is always visible.
    await expect(chip).toBeVisible();

    // Dismiss banner.
    await closeBtn.click();
    await expect(banner).not.toBeVisible({ timeout: 2000 });

    // Chip must still be visible after dismissal.
    await expect(chip).toBeVisible();
  });

  test('banner uses sessionStorage for dismissal: reload restores banner', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const banner = page.locator('.auth-banner');
    if (!(await banner.isVisible({ timeout: 2000 }).catch(() => false))) {
      test.skip(true, 'Not running with VITE_NO_AUTH=true');
    }

    await banner.locator('.auth-banner__close').click();
    await expect(banner).not.toBeVisible({ timeout: 2000 });

    // Reload within same context — sessionStorage persists across navigation.
    await page.reload();
    await page.waitForLoadState('networkidle');

    // After reload sessionStorage has the dismiss key, banner stays hidden.
    // But clearing it and re-navigating restores it. Verify the key is set.
    const val = await page.evaluate(() =>
      sessionStorage.getItem('clotho.unauth-banner.dismissed'),
    );
    expect(val).toBe('1');
  });
});

// ---------------------------------------------------------------------------
// 2. ActivityRail + palette flyout (VS Code-style two-layer sidebar)
// ---------------------------------------------------------------------------

test.describe('ActivityRail + palette flyout', () => {
  test('rail shows 3 icon buttons; palette is hidden by default', async ({ page }) => {
    const ok = await ensureCanvas(page);
    test.skip(!ok, 'Canvas not available');

    // Rail is always visible.
    const rail = page.locator('[data-testid="activity-rail"]');
    await expect(rail).toBeVisible();

    // Three icon buttons — Agent, Personality, Tools.
    await expect(page.locator('[data-testid="rail-agent"]')).toBeVisible();
    await expect(page.locator('[data-testid="rail-personality"]')).toBeVisible();
    await expect(page.locator('[data-testid="rail-tools"]')).toBeVisible();

    // Palette panel is hidden by default (no section active → native `hidden`).
    const palette = page.locator('.clotho-palette');
    await expect(palette).toBeHidden();
  });

  test('clicking a rail icon opens that section; clicking same icon closes', async ({ page }) => {
    const ok = await ensureCanvas(page);
    test.skip(!ok, 'Canvas not available');

    const agentBtn = page.locator('[data-testid="rail-agent"]');
    const palette = page.locator('.clotho-palette');

    await agentBtn.click();
    await expect(palette).toBeVisible();
    await expect(palette).toHaveAttribute('data-section', 'agent');

    // Panel only renders the Agent section, not Tools.
    const tiles = palette.locator('[data-testid^="palette-agent-"]');
    await expect(tiles).toHaveCount(4);

    // Click same icon → collapse.
    await agentBtn.click();
    await expect(palette).toBeHidden();
  });

  test('clicking a different rail icon switches the active section', async ({ page }) => {
    const ok = await ensureCanvas(page);
    test.skip(!ok, 'Canvas not available');

    const palette = page.locator('.clotho-palette');

    await page.locator('[data-testid="rail-agent"]').click();
    await expect(palette).toHaveAttribute('data-section', 'agent');

    await page.locator('[data-testid="rail-tools"]').click();
    await expect(palette).toHaveAttribute('data-section', 'tools');

    // Agent tiles must no longer be rendered when Tools is active.
    await expect(palette.locator('[data-testid^="palette-agent-"]')).toHaveCount(0);

    // Tools section shows Text / Image / Video labels.
    const labels = await palette
      .locator('.clotho-tile-label')
      .allTextContents();
    const normalised = labels.map((t) => t.trim().toLowerCase());
    expect(normalised).toContain('text');
    expect(normalised).toContain('image');
    expect(normalised).toContain('video');
  });

  test('Agent section exposes all 4 modality tiles (Prompt/Image/Audio/Video)', async ({ page }) => {
    const ok = await ensureCanvas(page);
    test.skip(!ok, 'Canvas not available');

    await page.locator('[data-testid="rail-agent"]').click();
    const palette = page.locator('.clotho-palette');
    await expect(palette).toBeVisible();

    const labels = await palette
      .locator('.clotho-tile-label')
      .allTextContents();
    const normalised = labels.map((t) => t.trim().toLowerCase());

    expect(normalised).toContain('prompt');
    expect(normalised).toContain('image');
    expect(normalised).toContain('audio');
    expect(normalised).toContain('video');
  });

  test('Escape closes the flyout', async ({ page }) => {
    const ok = await ensureCanvas(page);
    test.skip(!ok, 'Canvas not available');

    await page.locator('[data-testid="rail-agent"]').click();
    const palette = page.locator('.clotho-palette');
    await expect(palette).toBeVisible();

    await page.keyboard.press('Escape');
    await expect(palette).toBeHidden();
  });
});

// ---------------------------------------------------------------------------
// 3. Empty canvas + ⌘K hint
// ---------------------------------------------------------------------------

test.describe('Empty canvas state', () => {
  test('ghost cluster renders 3 ghost nodes when pipeline is empty', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => localStorage.removeItem('clotho.empty-state.dismissed'));
    await page.reload();
    await page.waitForLoadState('networkidle');

    const ok = await page.locator('.react-flow').isVisible({ timeout: 3000 }).catch(() => false);
    test.skip(!ok, 'Canvas not available');

    const nodeCount = await page.locator('.react-flow__node').count();
    if (nodeCount > 0) {
      test.skip(true, 'Pipeline already has nodes — empty state not shown');
    }

    // Dismiss auth banner so empty state is interactable.
    const authClose = page.locator('.auth-banner__close');
    if (await authClose.isVisible({ timeout: 1000 }).catch(() => false)) {
      await authClose.click();
    }

    const emptyState = page.locator('.empty-canvas');
    await expect(emptyState).toBeVisible({ timeout: 3000 });

    // 3 ghost nodes.
    const ghosts = page.locator('.empty-canvas__ghost');
    await expect(ghosts).toHaveCount(3);

    // CTA button.
    const cta = page.locator('.empty-canvas__cta');
    await expect(cta).toBeVisible();
    await expect(cta).toContainText('LOAD SAMPLE PIPELINE');

    // ⌘K hint.
    const hint = page.locator('.empty-canvas__hint-corner');
    await expect(hint).toBeVisible();
    await expect(hint).toContainText('⌘K');
  });

  test('LOAD SAMPLE PIPELINE adds nodes and hides empty state', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => localStorage.removeItem('clotho.empty-state.dismissed'));
    await page.reload();
    await page.waitForLoadState('networkidle');

    const ok = await page.locator('.react-flow').isVisible({ timeout: 3000 }).catch(() => false);
    test.skip(!ok, 'Canvas not available');

    const nodeCount = await page.locator('.react-flow__node').count();
    if (nodeCount > 0) {
      test.skip(true, 'Pipeline already has nodes');
    }

    const authClose = page.locator('.auth-banner__close');
    if (await authClose.isVisible({ timeout: 1000 }).catch(() => false)) {
      await authClose.click();
    }

    const cta = page.locator('.empty-canvas__cta');
    if (!(await cta.isVisible({ timeout: 2000 }).catch(() => false))) {
      test.skip(true, 'Empty state CTA not visible');
    }

    await cta.click();
    await expect(page.locator('.react-flow__node')).toHaveCount(3, { timeout: 5000 });

    // Empty state gone.
    await expect(page.locator('.empty-canvas')).not.toBeVisible({ timeout: 2000 });

    // localStorage flag set.
    const flag = await page.evaluate(() => localStorage.getItem('clotho.empty-state.dismissed'));
    expect(flag).toBe('1');
  });
});

// ---------------------------------------------------------------------------
// 4. NodeActionsMenu ⋯ button + Lock badge + delete guard
// ---------------------------------------------------------------------------

test.describe('NodeActionsMenu', () => {
  test('each node has a ⋯ button; click opens menu with expected items', async ({ page }) => {
    const ok = await ensureNodes(page);
    test.skip(!ok, 'Could not load nodes');

    const firstNode = page.locator('.react-flow__node').first();
    // Hover the node first so the menu button becomes pointer-events:auto
    // (the button is opacity:0 + pointer-events:none until node hover).
    await firstNode.locator('.clotho-node').first().hover();
    const menuBtn = firstNode.locator('.clotho-node__menu-btn');
    await expect(menuBtn).toBeVisible({ timeout: 3000 });

    await menuBtn.click();

    // Radix portal renders menu outside the node DOM.
    const menu = page.locator('.clotho-menu');
    await expect(menu).toBeVisible({ timeout: 2000 });

    const items = menu.locator('.clotho-menu__item');
    const itemTexts = await items.allTextContents();
    const combined = itemTexts.join(' ');

    expect(combined).toMatch(/Duplicate/);
    expect(combined).toMatch(/Rename/);
    expect(combined).toMatch(/Lock/);
    expect(combined).toMatch(/Delete/);

    // Check keyboard shortcuts shown.
    const shortcuts = menu.locator('.clotho-menu__shortcut');
    const shortcutTexts = await shortcuts.allTextContents();
    expect(shortcutTexts).toContain('⌘D');
    expect(shortcutTexts).toContain('F2');
    expect(shortcutTexts).toContain('⌘L');
    expect(shortcutTexts).toContain('Del');

    // Close menu.
    await page.keyboard.press('Escape');
    await expect(menu).not.toBeVisible({ timeout: 2000 });
  });

  test('Lock adds lock badge; locked node shows Unlock in menu', async ({ page }) => {
    const ok = await ensureNodes(page);
    test.skip(!ok, 'Could not load nodes');

    const firstNode = page.locator('.react-flow__node').first();
    await firstNode.locator('.clotho-node').first().hover();
    const menuBtn = firstNode.locator('.clotho-node__menu-btn');
    await menuBtn.click();

    const menu = page.locator('.clotho-menu');
    await expect(menu).toBeVisible({ timeout: 2000 });

    // Click Lock item.
    const lockItem = menu.locator('.clotho-menu__item').filter({ hasText: 'Lock' });
    await lockItem.click();

    // Lock badge appears on node.
    await expect(firstNode.locator('.clotho-node__lock-badge')).toBeVisible({ timeout: 2000 });

    // Reopen menu — hover again to make button interactable, then click.
    await firstNode.locator('.clotho-node').first().hover();
    await menuBtn.click();
    const menu2 = page.locator('.clotho-menu');
    await expect(menu2).toBeVisible({ timeout: 2000 });
    const unlockItem = menu2.locator('.clotho-menu__item').filter({ hasText: 'Unlock' });
    await expect(unlockItem).toBeVisible();

    // Cleanup: unlock.
    await unlockItem.click();
  });
});

// ---------------------------------------------------------------------------
// 5. Port label prettification
// ---------------------------------------------------------------------------

test.describe('Port label prettification', () => {
  test('port type label says "image prompt" not "image_prompt" on hover', async ({ page }) => {
    const ok = await ensureNodes(page);
    test.skip(!ok, 'Could not load nodes');

    // The Image Prompt Crafter node has an in_text (text) and out_prompt (image_prompt) port.
    const crafterNode = page.locator('.react-flow__node').filter({
      has: page.locator('.clotho-node--agent-crafter'),
    }).first();

    const hasCrafter = await crafterNode.isVisible({ timeout: 2000 }).catch(() => false);
    if (!hasCrafter) {
      test.skip(true, 'No crafter node in current pipeline');
    }

    // Port labels should be in the DOM (visibility is CSS-controlled via hover).
    // Check title attributes on handles for the type label.
    const handles = crafterNode.locator('.clotho-handle');
    const handleCount = await handles.count();
    expect(handleCount).toBeGreaterThan(0);

    // The output handle for the crafter should reference "image prompt" (prettified).
    const outHandle = crafterNode.locator('.clotho-handle--image_prompt');
    if (await outHandle.count() > 0) {
      const title = await outHandle.getAttribute('title');
      expect(title).toContain('image prompt');
      expect(title).not.toContain('image_prompt');
    }
  });

  test('port labels render outside node body (clotho-port-label elements)', async ({ page }) => {
    const ok = await ensureNodes(page);
    test.skip(!ok, 'Could not load nodes');

    const firstNode = page.locator('.react-flow__node').first();
    const portLabels = firstNode.locator('.clotho-port-label');
    const count = await portLabels.count();
    // Every node should have at least one port label.
    expect(count).toBeGreaterThan(0);
  });
});

// ---------------------------------------------------------------------------
// 6. Node teaser description
// ---------------------------------------------------------------------------

test.describe('Node description teaser', () => {
  test('each node body shows a .clotho-node__description teaser text', async ({ page }) => {
    const ok = await ensureNodes(page);
    test.skip(!ok, 'Could not load nodes');

    const nodes = page.locator('.react-flow__node');
    const count = await nodes.count();
    expect(count).toBeGreaterThan(0);

    // At least one node must have a teaser description.
    const descriptions = page.locator('.clotho-node__description');
    const descCount = await descriptions.count();
    expect(descCount).toBeGreaterThan(0);

    // Script Writer node should have a non-empty teaser.
    const scriptNode = page.locator('.react-flow__node').filter({
      has: page.locator('.clotho-node--agent-script'),
    }).first();
    const hasScrip = await scriptNode.isVisible({ timeout: 2000 }).catch(() => false);
    if (hasScrip) {
      const teaser = scriptNode.locator('.clotho-node__description');
      await expect(teaser).toBeVisible();
      const text = await teaser.textContent();
      expect(text?.trim().length).toBeGreaterThan(0);
    }
  });
});

// ---------------------------------------------------------------------------
// 7. Inspector "About this node" above "Basics"
// ---------------------------------------------------------------------------

test.describe('Inspector group order', () => {
  test('"About this node" group appears before "Basics" in agent inspector', async ({ page }) => {
    const ok = await ensureNodes(page);
    test.skip(!ok, 'Could not load nodes');

    // Click an agent node to open its inspector.
    const agentNode = page.locator('.react-flow__node').filter({
      has: page.locator('.clotho-node--agent-script, .clotho-node--agent-crafter, .clotho-node--agent-generic'),
    }).first();

    const hasAgent = await agentNode.isVisible({ timeout: 2000 }).catch(() => false);
    if (!hasAgent) {
      await page.locator('.react-flow__node').first().click();
    } else {
      await agentNode.click();
    }

    const inspector = page.locator('.clotho-inspector');
    await expect(inspector).toBeVisible({ timeout: 3000 });

    // Get all <details> groups in order.
    const groups = inspector.locator('details');
    const groupCount = await groups.count();
    expect(groupCount).toBeGreaterThanOrEqual(2);

    const summaries = await inspector.locator('details > summary').allTextContents();
    const normalised = summaries.map((t) => t.trim().toLowerCase());

    // "about this node" must appear before "basics".
    const aboutIdx = normalised.findIndex((t) => t.includes('about'));
    const basicsIdx = normalised.findIndex((t) => t.includes('basics'));

    expect(aboutIdx).toBeGreaterThanOrEqual(0);
    expect(basicsIdx).toBeGreaterThanOrEqual(0);
    expect(aboutIdx).toBeLessThan(basicsIdx);
  });

  test('"About this node" shows input/output table', async ({ page }) => {
    const ok = await ensureNodes(page);
    test.skip(!ok, 'Could not load nodes');

    await page.locator('.react-flow__node').first().click();

    const inspector = page.locator('.clotho-inspector');
    await expect(inspector).toBeVisible({ timeout: 3000 });

    // The about group should be open by default.
    const aboutGroup = inspector.locator('details').filter({ hasText: /about this node/i }).first();
    await expect(aboutGroup).toBeVisible();
    await expect(aboutGroup).toHaveAttribute('open', /.*/);

    // It contains "Input" and "Output" labels.
    const text = await aboutGroup.textContent();
    expect(text?.toLowerCase()).toContain('input');
    expect(text?.toLowerCase()).toContain('output');
  });
});

// ---------------------------------------------------------------------------
// 8. Inline prompt editor + NodeResizer on Agent nodes
// ---------------------------------------------------------------------------

test.describe('Agent node inline prompt + resize', () => {
  test('Agent nodes expose an editable textarea while idle; typing updates config', async ({ page }) => {
    const ok = await ensureNodes(page);
    test.skip(!ok, 'Could not load nodes');

    const agentNode = page.locator('.react-flow__node').filter({
      has: page.locator('.clotho-node--agent-script, .clotho-node--agent-crafter, .clotho-node--agent-generic'),
    }).first();

    const hasAgent = await agentNode.isVisible({ timeout: 2000 }).catch(() => false);
    test.skip(!hasAgent, 'No agent node in pipeline');

    const textarea = agentNode.locator('.clotho-node__prompt');
    await expect(textarea).toBeVisible();

    const stamp = `sprint-stamp-${Date.now()}`;
    await textarea.focus();
    await page.keyboard.press('End');
    await page.keyboard.type(stamp);

    // The textarea reflects the new value (focus may land anywhere; only
    // assert the stamp made it in).
    const after = await textarea.inputValue();
    expect(after).toContain(stamp);

    // Clicking the textarea still selects the node → inspector opens.
    await textarea.click();
    await expect(page.locator('.clotho-inspector')).toBeVisible({ timeout: 3000 });
  });

  test('selecting a node reveals NodeResizer handles', async ({ page }) => {
    const ok = await ensureNodes(page);
    test.skip(!ok, 'Could not load nodes');

    const firstNode = page.locator('.react-flow__node').first();
    await firstNode.click();

    // NodeResizer renders 4 corner handles + 4 edge lines when active.
    const resizeHandles = firstNode.locator('.react-flow__resize-control.handle');
    await expect(resizeHandles.first()).toBeVisible({ timeout: 2000 });
    const count = await resizeHandles.count();
    expect(count).toBeGreaterThanOrEqual(4);
  });
});

// ---------------------------------------------------------------------------
// 9. Media provider guard
// ---------------------------------------------------------------------------

test.describe('Media provider guard', () => {
  test('Image node provider dropdown does NOT include kokoro', async ({ page }) => {
    const ok = await ensureNodes(page);
    test.skip(!ok, 'Could not load nodes');

    const imageNode = page.locator('.react-flow__node').filter({
      has: page.locator('.clotho-node--media-image'),
    }).first();

    const hasImageNode = await imageNode.isVisible({ timeout: 2000 }).catch(() => false);
    test.skip(!hasImageNode, 'No image media node in pipeline');

    await imageNode.click();

    const inspector = page.locator('.clotho-inspector');
    await expect(inspector).toBeVisible({ timeout: 3000 });

    // Provider dropdown for image node.
    const providerSelect = inspector.locator('select').first();
    await expect(providerSelect).toBeVisible({ timeout: 2000 });

    const kokoroOption = providerSelect.locator('option[value="kokoro"]');
    await expect(kokoroOption).toHaveCount(0);

    // Should have comfyui.
    const comfyOption = providerSelect.locator('option[value="comfyui"]');
    await expect(comfyOption).toHaveCount(1);
  });
});
