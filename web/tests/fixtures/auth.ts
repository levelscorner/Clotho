import { test as base, expect, type Page } from '@playwright/test';

const ADMIN_EMAIL = 'admin@clotho.dev';
const ADMIN_PASSWORD = 'clotho123';

/** Login and store auth tokens in localStorage, returning an authenticated page. */
async function loginAsAdmin(page: Page) {
  await page.goto('/');
  // Wait for login form to appear
  await page.waitForSelector('input[type="email"]', { timeout: 10000 });
  await page.fill('input[type="email"]', ADMIN_EMAIL);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button[type="submit"]');
  // Wait for redirect to canvas (login disappears, canvas loads)
  await page.waitForSelector('.react-flow', { timeout: 15000 });
}

/** Fixture that provides a pre-authenticated page. */
export const test = base.extend<{ authedPage: Page }>({
  authedPage: async ({ page }, use) => {
    await loginAsAdmin(page);
    await use(page);
  },
});

export { expect, ADMIN_EMAIL, ADMIN_PASSWORD };
