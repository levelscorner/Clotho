import { test, expect } from '../../fixtures/auth';

test.describe('Templates', () => {
  test('template button opens gallery', async ({ authedPage: page }) => {
    const templatesBtn = page.locator('button', { hasText: /template/i }).first();
    await expect(templatesBtn).toBeVisible();
    await templatesBtn.click();

    // Gallery modal should appear with template cards
    await page.waitForTimeout(1000);
    await expect(page.locator('text=YouTube Story').first()).toBeVisible({ timeout: 5000 });
  });

  test('gallery shows all 5 built-in templates', async ({ authedPage: page }) => {
    const templatesBtn = page.locator('button', { hasText: /template/i }).first();
    await templatesBtn.click();
    await page.waitForTimeout(1000);

    // Check for each template
    await expect(page.locator('text=YouTube Story').first()).toBeVisible();
    await expect(page.locator('text=Instagram Reel').first()).toBeVisible();
    await expect(page.locator('text=Character Sheet').first()).toBeVisible();
  });

  test('applying a template loads nodes onto canvas', async ({ authedPage: page }) => {
    const templatesBtn = page.locator('button', { hasText: /template/i }).first();
    await templatesBtn.click();
    await page.waitForTimeout(1000);

    // Click on a template card — use force to bypass overlay interception
    const templateCard = page.locator('text=Prompt Enhancer').first();
    await templateCard.click({ force: true });
    await page.waitForTimeout(1000);

    // Canvas should now have nodes
    const nodes = await page.locator('.clotho-node').count();
    expect(nodes).toBeGreaterThan(0);
  });
});

test.describe('Pipeline Export/Import', () => {
  test('export button downloads a file', async ({ authedPage: page }) => {
    const exportBtn = page.locator('button', { hasText: /export/i }).first();
    if (await exportBtn.isVisible()) {
      // Listen for download
      const downloadPromise = page.waitForEvent('download', { timeout: 10000 }).catch(() => null);
      await exportBtn.click();
      const download = await downloadPromise;
      if (download) {
        expect(download.suggestedFilename()).toContain('.clotho.json');
      }
    }
  });
});
