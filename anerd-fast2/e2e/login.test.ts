import { expect, test } from '@playwright/test';

test('should display the login page', async ({ page }) => {
    await page.goto('/login');
    await expect(page.locator('h2')).toHaveText('Log in to GoFast');
});
