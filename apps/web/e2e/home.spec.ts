import { home } from './demo';
import { expect, test } from './fixtures';

test('the Home greets the user and lists what needs them', async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.goto('/');

  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Good morning, Alex.');
  await expect(page.getByText('Wednesday 21 October')).toBeVisible();
  await expect(page.getByText('3 things need you today. 1 is late.')).toBeVisible();
  await expect(page.getByRole('link', { name: /Nudge reviewers on WEB-12/ })).toBeVisible();
  await expect(page.getByText('09:41 · Tokyo')).toBeVisible();
  await expect(page).toHaveTitle('Bakhyard');
});

test('without a session, every page shows the Signed out screen', async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ status: 401 }));
  await page.goto('/');

  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Signed out');
  await expect(page.getByRole('link', { name: 'Sign in' })).toHaveAttribute('href', '/auth/google/login');
  await expect(page).toHaveTitle('Signed out · Bakhyard');
});
