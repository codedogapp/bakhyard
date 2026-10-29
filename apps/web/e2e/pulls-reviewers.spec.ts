import type { Locator } from '@playwright/test';

import { bounds } from './bounds';
import { home, pulls, reviewRequests } from './demo';
import { expect, test } from './fixtures';
import { answerPulls, answerReviews, mine, reviews } from './pulls';

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await answerPulls(page, pulls);
  await answerReviews(page, reviewRequests);
  await page.goto('/pulls');
});

function style(locator: Locator, property: string) {
  return locator.evaluate((element, name) => getComputedStyle(element).getPropertyValue(name), property);
}

test('each of my PRs says where it stands with its reviewers, under its repo', async ({ page }) => {
  // The review line is the second detail, right after the repo's (decision 174).
  await expect(mine(page).locator('.list-detail + .list-detail')).toHaveText([
    'Waiting for Leo Hart · Approved by Mia Chen',
    'No reviewers',
    'Changes requested by Noor Aziz · Waiting for Tom Berg and Leo Hart · Approved by Mia Chen',
  ]);
  await expect(
    mine(page).getByRole('link', { name: /^#128 .* acme\/web Waiting for Leo Hart · Approved by Mia Chen$/ })
  ).toHaveAttribute('href', 'https://bitbucket.org/acme/web/pull-requests/128');
  await expect(reviews(page).getByRole('link')).toHaveCount(2);
  await expect(reviews(page).locator('.list-detail')).toHaveText(['acme/web', 'acme/billing']);
});

test("the review line looks like the repo's, and starts under it", async ({ page }) => {
  const row = mine(page).getByRole('listitem').last();
  const [repo, line] = [row.getByText('acme/web', { exact: true }), row.getByText(/^Changes requested by/)];

  await expect(line).toHaveCSS('font-size', '14px');
  for (const property of ['color', 'font-family', 'font-size', 'font-weight', 'line-height', 'letter-spacing']) {
    expect(await style(line, property), property).toBe(await style(repo, property));
  }
  const [above, below] = [await bounds(repo), await bounds(line)];
  expect(below.x).toBeCloseTo(above.x, 1);
  expect(below.y).toBeCloseTo(above.y + above.height, 1);
});
