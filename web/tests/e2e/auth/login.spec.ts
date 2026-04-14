import { test, expect } from '@playwright/test';

const ADMIN_EMAIL = 'admin@clotho.dev';
const ADMIN_PASSWORD = 'clotho123';

/**
 * Authentication flow tests.
 * These are ONLY relevant when the server runs WITHOUT NO_AUTH mode.
 * When NO_AUTH=true, all tests in this file are automatically skipped.
 */
test.describe('Authentication', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');

    // If we're already in no-auth mode (canvas visible, no login form), skip the whole suite.
    const bannerVisible = await page.locator('.auth-banner').isVisible({ timeout: 1000 }).catch(() => false);
    if (bannerVisible) {
      test.skip(true, 'Running in NO_AUTH mode — login tests do not apply');
    }
  });

  test('shows login page when not authenticated', async ({ page }) => {
    await expect(page.locator('input[type="email"]')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('input[type="password"]')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('login with valid credentials shows canvas', async ({ page }) => {
    await page.fill('input[type="email"]', ADMIN_EMAIL);
    await page.fill('input[type="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');

    // Should redirect to canvas
    await expect(page.locator('.react-flow')).toBeVisible({ timeout: 15000 });
  });

  test('login with wrong password shows error', async ({ page }) => {
    await page.fill('input[type="email"]', ADMIN_EMAIL);
    await page.fill('input[type="password"]', 'wrongpassword');
    await page.click('button[type="submit"]');

    // Should stay on login page, no canvas.
    await page.waitForTimeout(2000);
    await expect(page.locator('.react-flow')).not.toBeVisible();
  });

  test('login persists across page reload', async ({ page }) => {
    await page.fill('input[type="email"]', ADMIN_EMAIL);
    await page.fill('input[type="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');
    await expect(page.locator('.react-flow')).toBeVisible({ timeout: 15000 });

    // Reload — should still be authenticated
    await page.reload();
    await expect(page.locator('.react-flow')).toBeVisible({ timeout: 15000 });
  });
});
