import type { Page } from '@playwright/test';

import { home } from './demo';
import { expect, test } from './fixtures';

test('the Home greets the user and lists what needs them', async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.goto('/');

  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Good morning, Alex.');
  await expect(page.getByText('Wednesday 21 October')).toBeVisible();
  await expect(page.getByText('3 things need you today. 1 is late.')).toBeVisible();
  await expect(page.getByRole('link', { name: /Nudge reviewers on WEB-12/ })).toBeVisible();
  await expect(page.getByRole('heading', { level: 2, name: 'This week' })).toBeVisible();
  await expect(page.getByText('09:41 · Tokyo')).toBeVisible();
  await expect(page).toHaveTitle('Bakhyard');
});

test('the Home leaves out This week when its answer has no week', async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: { ...home, week: undefined } }));
  await page.goto('/');

  await expect(page.getByText('3 things need you today. 1 is late.')).toBeVisible();
  await expect(page.getByRole('heading', { level: 2, name: 'Your apps' })).toBeVisible();
  await expect(page.getByRole('heading', { level: 2, name: 'This week' })).toHaveCount(0);
  await expect(page.getByRole('link', { name: 'Day report' })).toHaveCount(0);
  await expect(page.locator('main section').last()).toContainText('Your apps');
});

// The colour --ink-muted gives text in the page's theme.
function inkMuted(page: Page) {
  return page.evaluate(() => {
    const probe = document.body.appendChild(document.createElement('span'));
    probe.style.color = 'var(--ink-muted)';
    const { color } = getComputedStyle(probe);
    probe.remove();
    return color;
  });
}

test("the Home says the hours are unavailable when home couldn't get the week", async ({ page }, testInfo) => {
  await page.route('/api/home', (route) =>
    route.fulfill({ json: { ...home, week: undefined, weekState: 'unavailable' } })
  );
  await page.goto('/');

  const week = page.locator('main section').last();
  await expect(week.getByRole('heading', { level: 2 })).toHaveText('This week');
  await expect(week.getByRole('link', { name: 'Day report' })).toHaveAttribute('href', '/time');
  // In place of the hours, in their type and muted, without the label or the bar (decision 187).
  const hours = week.getByText('Hours unavailable');
  await expect(hours).toHaveCSS('font-size', testInfo.project.name === 'phone' ? '24px' : '28px');
  await expect(hours).toHaveCSS('font-weight', '500');
  await expect(hours).toHaveCSS('color', await inkMuted(page));
  await expect(week.getByText('Logged on Jira')).toHaveCount(0);
  await expect(week.locator('.track')).toHaveCount(0);
});

test('without a session, every page shows the Signed out screen', async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ status: 401 }));
  await page.goto('/');

  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Signed out');
  await expect(page.getByRole('link', { name: 'Sign in' })).toHaveAttribute('href', '/auth/google/login');
  await expect(page).toHaveTitle('Signed out · Bakhyard');
});
