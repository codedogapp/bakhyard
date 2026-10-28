import type { Page } from '@playwright/test';

import type { Snapshot } from '../src/pulls/pulls';
import { bounds } from './bounds';
import { chipLooks } from './chips';
import { home, pulls } from './demo';
import { expect, test } from './fixtures';
import { group } from './groups';

/** Answers the PRs page's call for my PRs with the snapshot, or with the status. */
async function answerPulls(page: Page, snapshot: Snapshot | number) {
  await page.route('/api/pulls/mine', (route) =>
    typeof snapshot === 'number' ? route.fulfill({ status: snapshot }) : route.fulfill({ json: snapshot })
  );
}

/** The PRs page's section of my PRs. */
function mine(page: Page) {
  return group(page, 'Your PRs');
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test('the PRs page lists my open pull requests on Bitbucket', async ({ page }) => {
  await answerPulls(page, pulls);
  await page.goto('/pulls');

  await expect(page).toHaveTitle('Pull requests · Bakhyard');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Pull requests');
  await expect(page.getByRole('link', { name: 'PRs', exact: true })).toHaveAttribute('aria-current', 'page');
  await expect(mine(page).getByText('3 PRs')).toBeVisible();
  const rows = mine(page).getByRole('link');
  await expect(rows).toHaveText([
    '#128 WEB-12 Show the order history on the account page acme/web',
    '#41 Retry the payment webhook after a timeout acme/payments Draft',
    '#126 WEB-9 Fix the date picker in Safari acme/web',
  ]);
  for (const [index, pull] of (pulls.pulls ?? []).entries()) {
    await expect(rows.nth(index)).toHaveAttribute('href', pull.url ?? '');
  }
  // A draft's chip keeps only its outline, like a ticket's chip that isn't in progress.
  expect(await chipLooks(mine(page).locator('.chip'))).toEqual(['outlined']);
});

test("a PR's row takes me to the PR on Bitbucket", async ({ page }) => {
  await answerPulls(page, pulls);
  await page.route('https://bitbucket.org/**', (route) =>
    route.fulfill({ contentType: 'text/html', body: '<title>acme/web #128</title>' })
  );
  await page.goto('/pulls');
  await mine(page)
    .getByRole('link', { name: /^#128 / })
    .click();

  await expect(page).toHaveURL('https://bitbucket.org/acme/web/pull-requests/128');
  await expect(page).toHaveTitle('acme/web #128');
});

test("a PR without a link to Bitbucket shows, but isn't a link", async ({ page }) => {
  const [first, ...rest] = pulls.pulls ?? [];
  await answerPulls(page, { ...pulls, pulls: [{ ...first, url: undefined }, ...rest] });
  await page.goto('/pulls');

  await expect(mine(page).getByRole('listitem')).toHaveCount(3);
  await expect(mine(page).getByRole('listitem').first()).toHaveText(
    '#128 WEB-12 Show the order history on the account page acme/web'
  );
  await expect(mine(page).getByRole('link')).toHaveCount(2);
});

test("a PR's row fills under the pointer only when it links to the PR", async ({ page }, testInfo) => {
  // A phone has no pointer to hover with.
  test.skip(testInfo.project.name === 'phone');
  const [first, ...rest] = pulls.pulls ?? [];
  await answerPulls(page, { ...pulls, pulls: [{ ...first, url: undefined }, ...rest] });
  await page.goto('/pulls');

  const unlinked = mine(page).getByRole('listitem').first().locator('.list-row');
  await unlinked.hover();
  await expect(unlinked).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');

  const link = mine(page).getByRole('link', { name: /^#41 / });
  await link.hover();
  await expect(link).not.toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
});

test('the PRs page loads my PRs again each time it shows', async ({ page }) => {
  let snapshot = pulls;
  await page.route('/api/pulls/mine', (route) => route.fulfill({ json: snapshot }));
  await page.goto('/pulls');
  await expect(mine(page).getByText('3 PRs')).toBeVisible();

  await page.getByRole('link', { name: 'Home', exact: true }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Good morning, Alex.' })).toBeVisible();
  // The pulls service finds a PR merged while the page is away.
  snapshot = { ...pulls, pulls: pulls.pulls?.slice(1) };
  await page.getByRole('link', { name: 'PRs', exact: true }).click();

  await expect(mine(page).getByText('2 PRs')).toBeVisible();
  await expect(mine(page).getByRole('link')).toHaveCount(2);
});

test('the PRs page says why the last search failed, and when its PRs are from', async ({ page }) => {
  await answerPulls(page, { ...pulls, failure: 'list my pull requests: 502 Bad Gateway' });
  await page.goto('/pulls');

  const line = mine(page).getByText(/^The last search failed/);
  await expect(line).toHaveText(
    'The last search failed: list my pull requests: 502 Bad Gateway. These PRs are from 21 October at 09:38.'
  );
  await expect(line.locator('time')).toHaveAttribute('datetime', '2026-10-21T00:38:00Z');
  await expect(mine(page).getByRole('link')).toHaveCount(3);
  const [above, card] = [await bounds(line), await bounds(mine(page).locator('.card'))];
  expect(card.y - (above.y + above.height)).toBeCloseTo(20, 1);
});

test('before a search succeeds, the PRs page says why it failed, with no list', async ({ page }) => {
  await answerPulls(page, { failure: 'list my pull requests: 401 Unauthorized' });
  await page.goto('/pulls');

  await expect(mine(page).getByText(/^The last search failed/)).toHaveText(
    'The last search failed: list my pull requests: 401 Unauthorized.'
  );
  await expect(mine(page).getByRole('list')).toHaveCount(0);
  await expect(mine(page).getByText(/^\d+ PRs?$/)).toHaveCount(0);
});

test('the PRs page says when I have no open pull request', async ({ page }) => {
  await answerPulls(page, { pulls: [], searched: '2026-10-21T00:38:00Z' });
  await page.goto('/pulls');

  await expect(mine(page).getByText('No open pull requests.')).toBeVisible();
  await expect(mine(page).getByText(/^\d+ PRs?$/)).toHaveCount(0);
});

test('the PRs page says when my PRs could not load', async ({ page }) => {
  await answerPulls(page, 500);
  await page.goto('/pulls');

  await expect(mine(page).getByText("Couldn't load the pull requests.")).toBeVisible();
  await expect(mine(page).getByRole('list')).toHaveCount(0);
});

test("the PRs page says when Bitbucket isn't set up, with no section", async ({ page }) => {
  await page.route('/api/pulls/mine', (route) => route.fulfill({ status: 404, body: 'Bitbucket is not set up\n' }));
  await page.goto('/pulls');

  await expect(page.getByRole('main').getByText("Bitbucket isn't set up.")).toBeVisible();
  await expect(page.getByRole('main').getByRole('heading', { level: 2 })).toHaveCount(0);
});

test('a long title, repo or failure wraps on a narrow screen', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 640 });
  const pull = {
    id: 1234567,
    title: `Read ${'x'.repeat(200)}`,
    repo: `acme/${'y'.repeat(60)}`,
    url: 'https://bitbucket.org/acme/web/pull-requests/1234567',
    draft: true,
  };
  const failure = `list my pull requests: Get "https://api.bitbucket.org/2.0/pullrequests/${'z'.repeat(200)}": EOF`;
  await answerPulls(page, { pulls: [pull], searched: '2026-10-21T00:38:00Z', failure });
  await page.goto('/pulls');

  await expect(mine(page).getByRole('link')).toHaveText(`#${pull.id} ${pull.title} ${pull.repo} Draft`);
  const scroll = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(scroll).toBe(0);
  // On a phone the number and the chip share the row's first line.
  const [label, chip] = [
    await mine(page).getByText(`#${pull.id}`).boundingBox(),
    await mine(page).locator('.chip').boundingBox(),
  ];
  expect(label && chip && label.x + label.width <= chip.x && Math.abs(label.y - chip.y) < label.height).toBe(true);
});
