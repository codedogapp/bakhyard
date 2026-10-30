import type { Item } from '../src/github/github';
import { chipLooks } from './chips';
import { home } from './demo';
import { expect, test } from './fixtures';
import { answerAllGitHub, answerGitHub, githubIssues, githubMine, githubReviews } from './github';
import { group } from './groups';

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test('the GitHub page lists my PRs, the review requests and my issues on GitHub', async ({ page }, testInfo) => {
  await answerAllGitHub(page);
  await page.goto('/github');

  await expect(page).toHaveTitle('GitHub · Bakhyard');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('GitHub');
  // The tab bar, the phone layout, has no tab for GitHub.
  const [name, current] = testInfo.project.name === 'phone' ? ['More', 'true'] : ['GitHub', 'page'];
  await expect(page.getByRole('link', { name, exact: true })).toHaveAttribute('aria-current', current);
  await expect(page.getByRole('main').getByRole('heading', { level: 2 })).toHaveText([
    'Your PRs',
    'Review requests',
    'Assigned issues',
  ]);
  const lists = [
    { head: 'Your PRs', count: '2 PRs', snapshot: githubMine },
    { head: 'Review requests', count: '1 PR', snapshot: githubReviews },
    { head: 'Assigned issues', count: '2 issues', snapshot: githubIssues },
  ];
  for (const { head, count, snapshot } of lists) {
    const items = snapshot.items ?? [];
    await expect(group(page, head).getByText(count)).toBeVisible();
    const rows = group(page, head).getByRole('link');
    await expect(rows).toHaveText(
      items.map((item) => `#${item.id} ${item.title} ${item.repo}${item.draft ? ' Draft' : ''}`)
    );
    for (const [index, item] of items.entries()) {
      await expect(rows.nth(index)).toHaveAttribute('href', item.url ?? '');
    }
  }
  // A draft's chip keeps only its outline, as on the PRs page.
  expect(await chipLooks(page.getByRole('main').locator('.chip'))).toEqual(['outlined']);
});

test('a row takes me to the issue on GitHub', async ({ page }) => {
  await answerAllGitHub(page);
  await page.route('https://github.com/**', (route) =>
    route.fulfill({ contentType: 'text/html', body: '<title>acme/cli #88</title>' })
  );
  await page.goto('/github');
  await group(page, 'Assigned issues').getByRole('link', { name: /^#88 / }).click();

  await expect(page).toHaveURL('https://github.com/acme/cli/issues/88');
  await expect(page).toHaveTitle('acme/cli #88');
});

test('a long title, repo or failure wraps on a narrow screen', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 640 });
  const issue: Item = {
    id: 1234567,
    title: `Read ${'x'.repeat(200)}`,
    repo: `acme/${'y'.repeat(60)}`,
    url: 'https://github.com/acme/web/issues/1234567',
    draft: false,
  };
  const failure = `search issues: Get "https://api.github.com/search/issues?q=${'z'.repeat(200)}": EOF`;
  await answerGitHub(page, 'mine', githubMine);
  await answerGitHub(page, 'reviews', githubReviews);
  await answerGitHub(page, 'issues', { items: [issue], searched: '2026-10-21T00:39:00Z', failure });
  await page.goto('/github');

  const issues = group(page, 'Assigned issues');
  await expect(issues.getByRole('link')).toHaveText(`#${issue.id} ${issue.title} ${issue.repo}`);
  await expect(issues.getByText(/^The last search failed/)).toHaveText(
    `The last search failed: ${failure}. These issues are from 21 October at 09:39.`
  );
  const scroll = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(scroll).toBe(0);
});
