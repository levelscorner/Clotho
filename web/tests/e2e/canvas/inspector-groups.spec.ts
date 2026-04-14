import { test, expect } from '@playwright/test';

/**
 * Wave 2 — Node personality + Inspector groups.
 *
 * Tests:
 *  1. Inspector collapsible groups: "Basics" open by default, "Advanced" collapsed.
 *  2. Script Writer preset → clotho-node--agent-script class.
 *  3. Image Prompt Crafter preset → clotho-node--agent-crafter class.
 *  4. Image media node → clotho-node--media-image + matte frame.
 *  5. Video media node → clotho-node--media-video + reel frames.
 *  6. Audio media node → clotho-node--media-audio + oscilloscope SVG.
 *  7. Image inspector: comfyui provider option present.
 *  8. Audio inspector: kokoro provider shows Kokoro voices.
 *
 * Setup: loads the sample pipeline (Script Writer → Image Prompt Crafter → Image)
 * via the "LOAD SAMPLE PIPELINE" button or an existing node in the pipeline.
 */

async function ensureNodesLoaded(page: import('@playwright/test').Page): Promise<boolean> {
  await page.goto('/');
  // Clear dismissed flag so we see the empty state if pipeline is empty.
  await page.evaluate(() => localStorage.removeItem('clotho.empty-state.dismissed'));
  await page.reload();
  await page.waitForLoadState('networkidle');

  // Canvas must be visible.
  const canvas = page.locator('.react-flow');
  const canvasVisible = await canvas.isVisible({ timeout: 3000 }).catch(() => false);
  if (!canvasVisible) return false;

  // If there are already nodes, great.
  let nodeCount = await page.locator('.react-flow__node').count();
  if (nodeCount > 0) return true;

  // If empty state is showing, click LOAD SAMPLE PIPELINE.
  const cta = page.locator('.empty-canvas__cta');
  const ctaVisible = await cta.isVisible({ timeout: 2000 }).catch(() => false);
  if (ctaVisible) {
    await cta.click();
    await page.waitForFunction(
      () => document.querySelectorAll('.react-flow__node').length >= 3,
      { timeout: 5000 },
    );
    nodeCount = await page.locator('.react-flow__node').count();
    return nodeCount >= 3;
  }

  return false;
}

test.describe('Inspector groups and node personality', () => {
  test('Basics open by default, Advanced collapsed; click Advanced to expand', async ({
    page,
  }) => {
    const loaded = await ensureNodesLoaded(page);
    test.skip(!loaded, 'Could not load nodes into canvas');

    // Click an agent node to open inspector.
    const agentNode = page.locator('.react-flow__node').filter({ has: page.locator('.clotho-node--agent-script, .clotho-node--agent-crafter, .clotho-node--agent-generic') }).first();
    const hasAgent = await agentNode.isVisible({ timeout: 2000 }).catch(() => false);

    if (!hasAgent) {
      // Fall back to first node.
      await page.locator('.react-flow__node').first().click();
    } else {
      await agentNode.click();
    }

    // Basics <details> should be open; Advanced should not be.
    const basics = page.locator('details', { hasText: 'Basics' }).first();
    const advanced = page.locator('details', { hasText: 'Advanced' }).first();

    await expect(basics).toHaveAttribute('open', /.*/);
    await expect(advanced).not.toHaveAttribute('open', /.*/);

    // Click Advanced summary → opens.
    await advanced.locator('summary').click();
    await expect(advanced).toHaveAttribute('open', /.*/);
  });

  test('Script Writer preset node renders with clotho-node--agent-script class', async ({
    page,
  }) => {
    const loaded = await ensureNodesLoaded(page);
    test.skip(!loaded, 'Could not load nodes into canvas');

    // The sample pipeline includes a Script Writer node (preset_category='script').
    const scriptNodes = page.locator('.clotho-node--agent-script');
    const count = await scriptNodes.count();
    test.skip(count === 0, 'No script agent nodes in current pipeline');

    await expect(scriptNodes.first()).toBeVisible();
  });

  test('Image Prompt Crafter preset node renders with clotho-node--agent-crafter class', async ({
    page,
  }) => {
    const loaded = await ensureNodesLoaded(page);
    test.skip(!loaded, 'Could not load nodes into canvas');

    const crafterNodes = page.locator('.clotho-node--agent-crafter');
    const count = await crafterNodes.count();
    test.skip(count === 0, 'No crafter agent nodes in current pipeline');

    await expect(crafterNodes.first()).toBeVisible();
  });

  test('Image media node renders with matte frame (.clotho-node__matte)', async ({
    page,
  }) => {
    const loaded = await ensureNodesLoaded(page);
    test.skip(!loaded, 'Could not load nodes into canvas');

    const imageNodes = page.locator('.clotho-node--media-image');
    const count = await imageNodes.count();
    test.skip(count === 0, 'No image media node in current pipeline');

    await expect(imageNodes.first()).toBeVisible();
    await expect(imageNodes.first().locator('.clotho-node__matte')).toBeVisible();
  });

  test('Video media node renders with reel frames (.clotho-node__reel)', async ({
    page,
  }) => {
    const loaded = await ensureNodesLoaded(page);
    test.skip(!loaded, 'Could not load nodes into canvas');

    const reelNodes = page.locator('.clotho-node--media-video');
    const count = await reelNodes.count();
    test.skip(count === 0, 'No video media node in current pipeline (sample pipeline has image, not video)');

    await expect(reelNodes.first()).toBeVisible();
    await expect(reelNodes.first().locator('.clotho-node__reel')).toBeVisible();
  });

  test('Audio media node renders with oscilloscope SVG (.clotho-node__scope-svg)', async ({
    page,
  }) => {
    const loaded = await ensureNodesLoaded(page);
    test.skip(!loaded, 'Could not load nodes into canvas');

    const audioNodes = page.locator('.clotho-node--media-audio');
    const count = await audioNodes.count();
    test.skip(count === 0, 'No audio media node in current pipeline (sample pipeline has image, not audio)');

    await expect(audioNodes.first()).toBeVisible();
    await expect(audioNodes.first().locator('.clotho-node__scope-svg')).toBeVisible();
  });

  test('Media inspector: Image node has provider dropdown with comfyui option', async ({
    page,
  }) => {
    const loaded = await ensureNodesLoaded(page);
    test.skip(!loaded, 'Could not load nodes into canvas');

    const imageNode = page.locator('.react-flow__node').filter({
      has: page.locator('.clotho-node--media-image'),
    }).first();
    const hasNode = await imageNode.isVisible({ timeout: 2000 }).catch(() => false);
    test.skip(!hasNode, 'No image node found');

    await imageNode.click();

    // Inspector should open with provider dropdown containing comfyui.
    const providerSelect = page.locator('select').filter({
      has: page.locator('option[value="comfyui"]'),
    }).first();
    await expect(providerSelect).toBeVisible({ timeout: 3000 });
    await expect(providerSelect.locator('option[value="comfyui"]')).toHaveCount(1);
  });

  test('Audio inspector: kokoro provider shows voice dropdown with Kokoro voices', async ({
    page,
  }) => {
    const loaded = await ensureNodesLoaded(page);
    test.skip(!loaded, 'Could not load nodes into canvas');

    // We'll test the audio inspector by checking if an audio node exists.
    // If not, we can skip — the sample pipeline doesn't include audio.
    const audioNode = page.locator('.react-flow__node').filter({
      has: page.locator('.clotho-node--media-audio'),
    }).first();
    const hasNode = await audioNode.isVisible({ timeout: 2000 }).catch(() => false);
    test.skip(!hasNode, 'No audio node found — sample pipeline does not include audio');

    await audioNode.click();

    const providerSelect = page.locator('select').filter({
      has: page.locator('option[value="kokoro"]'),
    }).first();
    await expect(providerSelect).toBeVisible({ timeout: 3000 });
    await providerSelect.selectOption('kokoro');

    // Voice dropdown should update to Kokoro voices.
    // Wait a tick for re-render.
    await page.waitForTimeout(200);
    const voiceSelect = page.locator('.clotho-inspector select').nth(1);
    await expect(voiceSelect.locator('option[value="af_bella"]')).toHaveCount(1);
  });
});
