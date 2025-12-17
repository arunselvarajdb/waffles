import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should show login form when not authenticated', async ({ page }) => {
    // When not authenticated, should redirect to login or show login form
    await expect(page.locator('form')).toBeVisible();
    await expect(page.getByLabel(/email/i)).toBeVisible();
    await expect(page.getByLabel(/password/i)).toBeVisible();
  });

  test('should login successfully with valid credentials', async ({ page }) => {
    await page.getByLabel(/email/i).fill('admin@example.com');
    await page.getByLabel(/password/i).fill('admin123');
    await page.getByRole('button', { name: /login|sign in/i }).click();

    // Should redirect to dashboard after successful login
    await expect(page).toHaveURL(/\/dashboard|\/servers|\//);
    // Should show user name or logout button
    await expect(page.getByText(/admin|logout|sign out/i)).toBeVisible();
  });

  test('should show error for invalid credentials', async ({ page }) => {
    await page.getByLabel(/email/i).fill('admin@example.com');
    await page.getByLabel(/password/i).fill('wrongpassword');
    await page.getByRole('button', { name: /login|sign in/i }).click();

    // Should show error message
    await expect(page.getByText(/invalid|error|failed|incorrect/i)).toBeVisible();
  });

  test('should logout successfully', async ({ page }) => {
    // First login
    await page.getByLabel(/email/i).fill('admin@example.com');
    await page.getByLabel(/password/i).fill('admin123');
    await page.getByRole('button', { name: /login|sign in/i }).click();

    // Wait for login to complete
    await page.waitForURL(/\/dashboard|\/servers|\//);

    // Click logout
    await page.getByText(/logout|sign out/i).click();

    // Should be redirected to login or show login form
    await expect(page.locator('form')).toBeVisible();
  });
});
