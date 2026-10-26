import { home } from './demo';
import { expect, test } from './fixtures';

test('the theme toggle goes from System to Light to Dark, and the choice outlives a reload', async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.goto('/');
  const root = page.locator('html');
  const toggle = page.getByRole('button', { name: /^Theme: / });

  await expect(toggle).toHaveAccessibleName('Theme: System');
  await expect(root).not.toHaveAttribute('data-theme');
  await expect(root).toHaveCSS('background-color', 'rgb(250, 250, 248)');

  await toggle.click();
  await expect(toggle).toHaveAccessibleName('Theme: Light');
  await expect(root).toHaveAttribute('data-theme', 'light');

  await toggle.click();
  await expect(toggle).toHaveAccessibleName('Theme: Dark');
  await expect(root).toHaveCSS('background-color', 'rgb(19, 19, 18)');

  await page.reload();
  await expect(toggle).toHaveAccessibleName('Theme: Dark');
  await expect(root).toHaveCSS('background-color', 'rgb(19, 19, 18)');
});
