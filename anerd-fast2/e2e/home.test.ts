import { test, expect } from '@playwright/test';

test('should display home page after authentication', async ({ page }) => {
  // The auth.setup.ts script handles authentication.
  // This test starts as an authenticated user.
  await page.goto('/');

  // Verify that the home page content is visible
  await expect(page.locator('h1')).toHaveText('GoFast v2');
});
