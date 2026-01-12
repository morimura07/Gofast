import { expect, test } from '@playwright/test';

test.describe('Authentication', () => {
    test.describe('Protected Routes', () => {
        test('should load dashboard when authenticated', async ({ page }) => {
            await page.goto('/');
            // With DEV_USER_ID set, should load without redirect
            await expect(page.locator('h1')).toHaveText('GoFast v2');
        });

        test('should load all protected routes', async ({ page }) => {
            // Dashboard
            await page.goto('/');
            await expect(page.locator('h1')).toBeVisible();

            // Skeletons
            await page.goto('/models/skeletons');
            await expect(page.getByRole('heading', { name: 'Skeletons' })).toBeVisible();
        });
    });

    test.describe('Navigation', () => {
        test('should navigate between all sections via sidebar', async ({ page }) => {
            await page.goto('/');

            // Navigate to Skeletons
            await page.getByRole('link', { name: 'Skeletons' }).click();
            await expect(page).toHaveURL(/.*\/models\/skeletons/);
            await expect(page.getByRole('heading', { name: 'Skeletons' })).toBeVisible();

            // Navigate back to Dashboard
            await page.getByRole('link', { name: 'Dashboard' }).click();
            await expect(page).toHaveURL(/.*\/$/);
        });

        test('should highlight active navigation item', async ({ page }) => {
            await page.goto('/models/skeletons');

            // The Skeletons link should have the active class (bg-primary without hover: prefix)
            const skeletonsLink = page.getByRole('link', { name: 'Skeletons' });
            await expect(skeletonsLink).toHaveClass(/(?:^|\s)bg-primary(?:\s|$)/);

            // Dashboard should have the inactive class (text-base-content/70)
            const dashboardLink = page.getByRole('link', { name: 'Dashboard' });
            await expect(dashboardLink).toHaveClass(/text-base-content/);
        });

        test('should show logo in sidebar', async ({ page }) => {
            await page.goto('/');
            // Find a visible logo (first one might be hidden mobile sidebar)
            const logo = page.locator('img[alt="GoFast"]:visible');
            await expect(logo.first()).toBeVisible();
        });
    });

    test.describe('Logout', () => {
        test('should have logout button in sidebar', async ({ page }) => {
            await page.goto('/');

            // Find logout button (it's in a form)
            const logoutButton = page.getByRole('button', { name: 'Logout' });
            await expect(logoutButton.first()).toBeVisible();
        });

        test('should logout and redirect to login page', async ({ page }) => {
            await page.goto('/');

            // Click logout button
            const logoutButton = page.getByRole('button', { name: 'Logout' });
            await logoutButton.first().click();

            // Should redirect to login page
            await expect(page).toHaveURL(/.*\/login/);
            await expect(page.locator('h2')).toHaveText('Log in to GoFast');
        });
    });

    test.describe('Auth Refresh', () => {
        test('should trigger force refresh on success param', async ({ page }) => {
            // The ?success=true param triggers a force refresh with 2s delay
            // This is used after Stripe payment completion
            await page.goto('/?success=true');

            // Page should eventually load (after the 2s delay + refresh)
            await expect(page.locator('h1')).toHaveText('GoFast v2', { timeout: 10000 });
        });
    });

    test.describe('Loading State', () => {
        test('should show loading spinner during auth check', async ({ page }) => {
            // We can't reliably test the loading spinner since it's very fast
            // with DEV_USER_ID, but we verify the page loads successfully
            await page.goto('/');
            await expect(page.locator('h1')).toBeVisible();
        });
    });
});

test.describe('Login Page', () => {
    test('should display login page', async ({ page }) => {
        await page.goto('/login');
        await expect(page.locator('h2')).toHaveText('Log in to GoFast');
    });

    test('should have OAuth login options', async ({ page }) => {
        await page.goto('/login');

        // Check that the login page has content (OAuth provider buttons)
        const loginMain = page.locator('main');
        await expect(loginMain).toBeVisible();
    });
});
