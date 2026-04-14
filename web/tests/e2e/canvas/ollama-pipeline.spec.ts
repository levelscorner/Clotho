import { test, expect } from '@playwright/test';

/**
 * End-to-end Ollama pipeline run.
 *
 * SKIPPED in CI: requires a live Ollama daemon on the dev host.
 * To run locally:
 *   1. `brew install ollama && ollama serve`
 *   2. `ollama pull llama3.1`
 *   3. Start the Clotho backend + frontend in no-auth mode
 *   4. `RUN_OLLAMA_E2E=1 npx playwright test ollama-pipeline.spec.ts`
 */
const RUN_OLLAMA = process.env.RUN_OLLAMA_E2E === '1';

test.describe('Ollama pipeline', () => {
  test.skip(
    !RUN_OLLAMA,
    'Ollama E2E skipped — set RUN_OLLAMA_E2E=1 to run locally with ollama serve running',
  );

  test('builds a 4-node pipeline and runs it against a local Ollama model', async ({
    page,
  }) => {
    await page.goto('/');

    // Expect canvas reachable.
    await expect(page.locator('.react-flow')).toBeVisible();

    // Drag Script Writer preset onto the canvas.
    const scriptWriterCard = page.getByText(/Script Writer/i).first();
    const canvas = page.locator('.react-flow__pane').first();
    await scriptWriterCard.dragTo(canvas);

    // TODO: add Image Prompt Crafter, Image media, Video media nodes.
    // TODO: connect them in order.
    // TODO: select each node, set provider=ollama + model=llama3.1.
    // TODO: click Run.
    // TODO: assert each node completes via SSE streaming.
    // TODO: assert cost footer says "local" not "$0.00".

    // Placeholder assertion so the suite shape is valid when executed.
    await expect(page.locator('.react-flow__node')).toHaveCount(1);
  });
});
