import { bounds } from './bounds';
import { home, pulls, reviewRequests } from './demo';
import { expect, test } from './fixtures';
import { heads } from './groups';
import { answerPulls, answerReviews, mine, reviews } from './pulls';

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test('the PRs page lists the PRs that wait for my review, under mine', async ({ page }, testInfo) => {
  await answerPulls(page, pulls);
  await answerReviews(page, reviewRequests);
  await page.goto('/pulls');

  await expect(heads(page)).toHaveText(['Your PRs', 'Review requests']);
  await expect(reviews(page).getByText('2 PRs')).toBeVisible();
  const rows = reviews(page).getByRole('link');
  await expect(rows).toHaveText([
    '#131 WEB-14 Cache the product images at the edge acme/web',
    '#87 Add the monthly invoice export acme/billing',
  ]);
  for (const [index, pull] of (reviewRequests.pulls ?? []).entries()) {
    await expect(rows.nth(index)).toHaveAttribute('href', pull.url ?? '');
  }
  // The sections stack as every page's do, a section's space under my PRs' card.
  const [card, section] = [await bounds(mine(page).locator('.card')), await bounds(reviews(page))];
  expect(section.y - (card.y + card.height)).toBeCloseTo(testInfo.project.name === 'phone' ? 48 : 64, 1);
});

test('my PRs show while the review requests load, which then show under them without moving them', async ({ page }) => {
  await answerPulls(page, pulls);
  const held = Promise.withResolvers<void>();
  await page.route('/api/pulls/reviews', async (route) => {
    await held.promise;
    await route.fulfill({ json: reviewRequests });
  });
  await page.goto('/pulls');

  await expect(mine(page).getByText('3 PRs')).toBeVisible();
  await expect(heads(page)).toHaveText(['Your PRs']);
  const before = await bounds(mine(page));
  held.resolve();

  await expect(heads(page)).toHaveText(['Your PRs', 'Review requests']);
  expect(await bounds(mine(page))).toEqual(before);
});

test('the PRs page loads the review requests again each time it shows', async ({ page }) => {
  let snapshot = reviewRequests;
  await answerPulls(page, pulls);
  await page.route('/api/pulls/reviews', (route) => route.fulfill({ json: snapshot }));
  await page.goto('/pulls');
  await expect(reviews(page).getByText('2 PRs')).toBeVisible();

  await page.getByRole('link', { name: 'Home', exact: true }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Good morning, Alex.' })).toBeVisible();
  // I review a PR while the page is away, and the pulls service no longer finds it.
  snapshot = { ...reviewRequests, pulls: reviewRequests.pulls?.slice(1) };
  await page.getByRole('link', { name: 'PRs', exact: true }).click();

  await expect(reviews(page).getByText('1 PR', { exact: true })).toBeVisible();
  await expect(reviews(page).getByRole('link')).toHaveText(['#87 Add the monthly invoice export acme/billing']);
});

test('the PRs page says when no PR waits for my review', async ({ page }) => {
  await answerPulls(page, pulls);
  await answerReviews(page, { pulls: [], searched: '2026-10-21T00:30:00Z' });
  await page.goto('/pulls');

  await expect(reviews(page).getByText('No pull requests to review.')).toBeVisible();
  await expect(reviews(page).getByText(/^\d+ PRs?$/)).toHaveCount(0);
});

test('the review requests say why their last search failed, and when they are from', async ({ page }) => {
  await answerPulls(page, pulls);
  await answerReviews(page, { ...reviewRequests, failure: 'list the Bitbucket repositories: 429 Too Many Requests' });
  await page.goto('/pulls');

  const line = reviews(page).getByText(/^The last search failed/);
  await expect(line).toHaveText(
    'The last search failed: list the Bitbucket repositories: 429 Too Many Requests. These PRs are from 21 October at 09:30.'
  );
  await expect(line.locator('time')).toHaveAttribute('datetime', '2026-10-21T00:30:00Z');
  await expect(reviews(page).getByRole('link')).toHaveCount(2);
  await expect(mine(page).getByText(/failed/)).toHaveCount(0);
});

test('the PRs page says when the review requests could not load, beside my PRs', async ({ page }) => {
  await answerPulls(page, pulls);
  await answerReviews(page, 500);
  await page.goto('/pulls');

  await expect(reviews(page).getByText("Couldn't load the review requests.")).toBeVisible();
  await expect(mine(page).getByRole('link')).toHaveCount(3);
  await expect(heads(page)).toHaveText(['Your PRs', 'Review requests']);
});

test('a pulls service older than the review requests shows my PRs alone', async ({ page }) => {
  // It answers 404 for the review requests, as chi does for a path it doesn't serve,
  // before my PRs come in.
  const answered = page.waitForResponse('**/api/pulls/reviews');
  await page.route('/api/pulls/reviews', (route) => route.fulfill({ status: 404, body: '404 page not found\n' }));
  await page.route('/api/pulls/mine', async (route) => {
    await answered;
    await route.fulfill({ json: pulls });
  });
  await page.goto('/pulls');

  await expect(mine(page).getByText('3 PRs')).toBeVisible();
  await expect(heads(page)).toHaveText(['Your PRs']);
  await expect(page.getByRole('main').getByText("Bitbucket isn't set up.")).toHaveCount(0);
});
