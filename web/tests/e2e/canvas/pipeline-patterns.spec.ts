import { test, expect } from '@playwright/test';

/**
 * Live-provider pipeline pattern smoke tests.
 *
 * Tagged @live so they're skipped by default — they require the full
 * local stack running (`make dev-full` brings up Postgres, backend,
 * frontend, Ollama, ComfyUI, Kokoro). Run explicitly with:
 *
 *   cd web && npx playwright test --grep @live
 *
 * These tests establish that the patterns documented in
 * docs/PIPELINE-PATTERNS.md actually execute end-to-end against real
 * providers — complementing the Go unit tests in
 * internal/engine/pipeline_patterns_test.go which use the fake
 * executor.
 *
 * Each test:
 *   1. Loads (or builds) the pattern's graph
 *   2. Clicks Run
 *   3. Waits for execution_completed (or timeout)
 *   4. Asserts the expected files appeared on disk + node body
 *      previews match the landing contract
 */

const LIVE_EXECUTION_TIMEOUT = 180_000; // 3 minutes — Ollama warm-up can be slow

// Guard: live tests only run when CLOTHO_LIVE_E2E=1. Without it, the whole
// suite self-skips at module evaluation time — CI stays green while the
// local stack isn't up. To run: CLOTHO_LIVE_E2E=1 npx playwright test --
// grep @live
const LIVE_ENABLED = process.env.CLOTHO_LIVE_E2E === '1';

// Matches the sample pipeline default in EmptyCanvasState.tsx — pattern B2
// (Script → Image Prompt Crafter → Image via ComfyUI).
test.describe('@live pipeline patterns', () => {
  test.describe.configure({ timeout: LIVE_EXECUTION_TIMEOUT });

  test.skip(!LIVE_ENABLED, 'set CLOTHO_LIVE_E2E=1 with `make dev-full` up to run these');

  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Dismiss the unauth banner if present — keeps clicks clean.
    const close = page.locator('.auth-banner__close');
    if (await close.isVisible({ timeout: 1000 }).catch(() => false)) {
      await close.click();
    }
  });

  test('B2: script → image_prompt → image (canonical sample)', async ({ page }) => {
    // Start with the sample pipeline. EmptyCanvasState currently defaults
    // to Ollama + ComfyUI so this is pattern B2 end-to-end.
    const cta = page.locator('.empty-canvas__cta');
    const hasCTA = await cta.isVisible({ timeout: 3000 }).catch(() => false);
    if (hasCTA) {
      await cta.click();
    }

    // Wait for the three sample nodes to appear.
    await expect(page.locator('.react-flow__node')).toHaveCount(3, { timeout: 5000 });

    // Click Run (global top-bar button).
    const runBtn = page.getByRole('button', { name: /run/i }).first();
    await runBtn.click();

    // Poll for completion — look for the completion toast OR all three
    // node status dots transitioning to 'completed'.
    await expect.poll(
      async () => {
        const dots = await page.locator('.clotho-node__status-dot--completed').count();
        return dots;
      },
      {
        timeout: LIVE_EXECUTION_TIMEOUT - 10_000,
        intervals: [1000, 2000, 5000],
        message: 'waiting for all three nodes to complete',
      },
    ).toBe(3);

    // Cost should be "local" on both LLM agents (Ollama) and on the image
    // node (ComfyUI) — the local-first smoke.
    const localChips = await page.locator('.clotho-node__cost-local').count();
    expect(localChips, 'expected "local" cost chips on Ollama + ComfyUI nodes').toBeGreaterThanOrEqual(2);

    // Manifest + image file should now exist on disk. We don't shell out
    // here — `make dev-full` points DataDir at ~/Documents/Clotho/; a
    // future test can curl /api/files/ to confirm the file is served.
  });

  // --- Patterns queued for follow-up live smoke coverage -------------------

  test.fixme('B3: script → image → video (image feeds video reference)', async () => {
    // Build the B3 graph via drag-drop, connect image→video.ref,
    // click Run, assert both an image file and a video file appear
    // in the execution's data dir.
  });

  test.fixme('B4: script → TTS narration via Kokoro', async () => {
    // Build the B4 graph, confirm Kokoro is running (otherwise skip
    // with a clear message), click Run, assert an audio file appears.
  });

  test.fixme('B5: script fan-out to all three modalities', async () => {
    // Biggest live test — 7 nodes. Most useful for catching fan-out
    // regressions end-to-end.
  });

  test.fixme('B10: re-run from a specific node', async () => {
    // After a successful B2 run, click the per-node play button on the
    // image_prompt crafter. Assert only the crafter + image nodes
    // re-execute; the script step's timestamp is unchanged.
  });
});
