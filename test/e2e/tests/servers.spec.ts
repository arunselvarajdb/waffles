import { test, expect } from '@playwright/test';

test.describe('Server Management', () => {
  test.beforeEach(async ({ page }) => {
    // Login before each test
    await page.goto('/');
    await page.getByLabel(/email/i).fill('admin@example.com');
    await page.getByLabel(/password/i).fill('admin123');
    await page.getByRole('button', { name: /login|sign in/i }).click();
    await page.waitForURL(/\/dashboard|\/servers|\//);
  });

  test('should display servers list', async ({ page }) => {
    // Navigate to servers page
    await page.getByRole('link', { name: /servers/i }).click();

    // Should show servers list or table
    await expect(page.getByRole('table').or(page.locator('[data-testid="servers-list"]'))).toBeVisible();
  });

  test('should open add server dialog', async ({ page }) => {
    await page.getByRole('link', { name: /servers/i }).click();

    // Click add server button
    await page.getByRole('button', { name: /add.*server|new.*server|register/i }).click();

    // Should show form dialog/modal
    await expect(page.getByRole('dialog').or(page.locator('form'))).toBeVisible();
    await expect(page.getByLabel(/name/i)).toBeVisible();
    await expect(page.getByLabel(/url/i)).toBeVisible();
  });

  test('should create a new server', async ({ page }) => {
    await page.getByRole('link', { name: /servers/i }).click();

    // Click add server button
    await page.getByRole('button', { name: /add.*server|new.*server|register/i }).click();

    // Fill in server details
    await page.getByLabel(/name/i).fill('Test Server E2E');
    await page.getByLabel(/description/i).fill('Created by E2E test');
    await page.getByLabel(/url/i).fill('http://mock-mcp:9001');

    // Select transport if dropdown exists
    const transportSelect = page.getByLabel(/transport/i);
    if (await transportSelect.isVisible()) {
      await transportSelect.selectOption('http');
    }

    // Submit
    await page.getByRole('button', { name: /create|save|submit|register/i }).click();

    // Should show success message or redirect
    await expect(page.getByText(/success|created|saved/i).or(page.getByText('Test Server E2E'))).toBeVisible();
  });

  test('should view server details', async ({ page }) => {
    await page.getByRole('link', { name: /servers/i }).click();

    // Click on first server row/card
    await page.locator('tr, [data-testid="server-card"]').first().click();

    // Should show server details
    await expect(page.getByText(/details|configuration|settings/i)).toBeVisible();
  });

  test('should delete a server', async ({ page }) => {
    await page.getByRole('link', { name: /servers/i }).click();

    // Find delete button on first server
    const deleteButton = page.getByRole('button', { name: /delete|remove/i }).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();

      // Confirm deletion if dialog appears
      const confirmButton = page.getByRole('button', { name: /confirm|yes|delete/i });
      if (await confirmButton.isVisible()) {
        await confirmButton.click();
      }

      // Should show success message
      await expect(page.getByText(/deleted|removed|success/i)).toBeVisible();
    }
  });
});
