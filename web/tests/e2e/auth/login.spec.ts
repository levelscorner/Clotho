import { test, expect } from '@playwright/test';

const ADMIN_EMAIL = 'admin@clotho.dev';
const ADMIN_PASSWORD = 'clotho123';

test.describe('Authentication', () => {
  test.beforeEach(async ({ page }) => {
    // Clear any stored tokens
    await page.goto('/');
    await page.evaluate(() => localStorage.clear());
    await page.reload();
  });

  test('shows login page when not authenticated', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('input[type="email"]')).toBeVisible();
    await expect(page.locator('input[type="password"]')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('login with valid credentials shows canvas', async ({ page }) => {
    await page.goto('/');
    await page.fill('input[type="email"]', ADMIN_EMAIL);
    await page.fill('input[type="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');

    // Should redirect to canvas
    await expect(page.locator('.react-flow')).toBeVisible({ timeout: 15000 });
  });

  test('login with wrong password shows error', async ({ page }) => {
    await page.goto('/');
    await page.fill('input[type="email"]', ADMIN_EMAIL);
    await page.fill('input[type="password"]', 'wrongpassword');
    await page.click('button[type="submit"]');

    // Should show error message, not redirect
    await page.waitForTimeout(2000);
    await expect(page.locator('.react-flow')).not.toBeVisible();
  });

  test('login persists across page reload', async ({ page }) => {
    await page.goto('/');
    await page.fill('input[type="email"]', ADMIN_EMAIL);
    await page.fill('input[type="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');
    await expect(page.locator('.react-flow')).toBeVisible({ timeout: 15000 });

    // Reload — should still be authenticated
    await page.reload();
    await expect(page.locator('.react-flow')).toBeVisible({ timeout: 15000 });
  });
});
