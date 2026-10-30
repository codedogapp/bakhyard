import { home, plans } from './demo';
import { expect, test } from './fixtures';
import { group } from './groups';

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test('the Planner page lists the plans, the last updated first', async ({ page }, testInfo) => {
  await page.route('/api/planner/plans', (route) => route.fulfill({ json: { plans } }));
  await page.goto('/planner');

  await expect(page).toHaveTitle('Planner · Bakhyard');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Planner');
  // The tab bar, the phone layout, has no tab for the Planner.
  const [name, current] = testInfo.project.name === 'phone' ? ['More', 'true'] : ['Planner', 'page'];
  await expect(page.getByRole('link', { name, exact: true })).toHaveAttribute('aria-current', current);
  const yours = group(page, 'Your plans');
  await expect(yours.getByText('4 plans')).toBeVisible();
  await expect(yours.getByRole('listitem')).toHaveText([
    'Today Add a dark theme to the docs site',
    'Yesterday Move the nightly backups to object storage',
    '17 Oct Split the settings page into tabs',
    "12 Oct Plan how to move the team's wiki to the docs site. Keep the old links working,…",
  ]);
  await expect(yours.getByText('Today')).toHaveAttribute('datetime', '2026-10-21T00:32:00Z');
  await expect(page.getByRole('main').getByRole('link')).toHaveCount(0);
});

test('the Planner page says when there is no plan yet', async ({ page }) => {
  await page.route('/api/planner/plans', (route) => route.fulfill({ json: { plans: [] } }));
  await page.goto('/planner');

  await expect(group(page, 'Your plans').getByText('No plans yet.')).toBeVisible();
});
