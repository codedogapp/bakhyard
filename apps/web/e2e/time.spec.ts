import type { Locator, Page } from '@playwright/test';

import type { Report } from '../src/time/report';
import { chipLooks } from './chips';
import { home, report, tickets } from './demo';
import { expect, test } from './fixtures';
import { answerTickets } from './jira';

/** Answers the Time page's call for the report with the report, or with the status. */
async function answerReport(page: Page, answer: Report | number) {
  await page.route('/api/time/report', (route) =>
    typeof answer === 'number' ? route.fulfill({ status: answer }) : route.fulfill({ json: answer })
  );
}

const heads = (page: Page) => page.getByRole('main').getByRole('heading', { level: 2 });
const days = (page: Page) => page.getByRole('main').getByRole('listitem');
const color = (locator: Locator) => locator.evaluate((element) => getComputedStyle(element).color);

// My open tickets, for the form that fills a day (time-fill.spec.ts).
test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await answerTickets(page, tickets);
});

test("the Time page shows this week's hours and the last 7 days' from Jira", async ({ page }) => {
  await answerReport(page, report);
  await page.goto('/time');

  await expect(page).toHaveTitle('Time · Bakhyard');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Time');
  await expect(heads(page)).toHaveText(['This week', 'Last 7 days']);
  await expect(page.getByText('15h 45m / 40h')).toBeVisible();
  await expect(page.getByText('1 day missing')).toBeVisible();
  await expect(days(page)).toHaveText([
    '15 Oct Thursday 7h 30m',
    '16 Oct Friday Missing 0h',
    '17 Oct Saturday 0h',
    '18 Oct Sunday 0h',
    '19 Oct Monday 8h',
    '20 Oct Tuesday 6h 15m',
    '21 Oct Today 1h 30m',
  ]);
  // Missing reads like a late item's chip.
  expect(await chipLooks(page.getByRole('main').locator('.chip'))).toEqual(['filled']);
});

test('the hours of a day show in the ink once logged, and stay muted like its date when not', async ({ page }) => {
  await answerReport(page, report);
  await page.goto('/time');

  const hours = (day: string) => days(page).filter({ hasText: day }).locator('.list-end > :last-child');
  await expect(hours('19 Oct')).toHaveText('8h');
  expect(await color(hours('19 Oct'))).toBe(await color(days(page).getByText('Monday')));
  expect(await color(hours('17 Oct'))).toBe(await color(days(page).getByText('17 Oct')));
  expect(await color(hours('17 Oct'))).not.toBe(await color(days(page).getByText('Saturday')));
});

test('the Time page is current in the header, or under More in the tab bar', async ({ page }, testInfo) => {
  await answerReport(page, report);
  await page.goto('/time');

  // The tab bar, the phone layout, has no tab for Time.
  const [name, current] = testInfo.project.name === 'phone' ? ['More', 'true'] : ['Time', 'page'];
  await expect(page.getByRole('link', { name, exact: true })).toHaveAttribute('aria-current', current);
});

test("the Home's day report link opens the Time page, which loads the report each time it shows", async ({ page }) => {
  let answer = report;
  await page.route('/api/time/report', (route) => route.fulfill({ json: answer }));
  await page.goto('/time');
  await expect(days(page).last()).toHaveText('21 Oct Today 1h 30m');

  await page.getByRole('link', { name: 'Home', exact: true }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Good morning, Alex.' })).toBeVisible();
  // The time service finds an hour logged while the page is away.
  answer = {
    ...report,
    days: report.days?.map((day) => (day.date === '2026-10-21' ? { ...day, loggedHours: 2.5 } : day)),
    week: { loggedHours: 16.75, targetHours: 40 },
  };
  await page.getByRole('link', { name: 'Day report' }).click();

  await expect(page).toHaveURL('/time');
  await expect(days(page).last()).toHaveText('21 Oct Today 2h 30m');
  await expect(page.getByText('16h 45m / 40h')).toBeVisible();
});

test('the Time page says why the last search failed, and when its hours are from', async ({ page }, testInfo) => {
  await answerReport(page, { ...report, failure: 'search my worklogs: 502 Bad Gateway' });
  await page.goto('/time');

  const line = page.getByRole('main').getByText(/^The last search failed/);
  await expect(line).toHaveText(
    'The last search failed: search my worklogs: 502 Bad Gateway. These hours are from 21 October at 09:38.'
  );
  await expect(line.locator('time')).toHaveAttribute('datetime', '2026-10-21T00:38:00Z');
  await expect(heads(page)).toHaveText(['This week', 'Last 7 days']);
  // Over the sections, the line leaves the space that the Todos page's form leaves over its first section.
  await expect(line).toHaveCSS('padding-bottom', testInfo.project.name === 'phone' ? '24px' : '30px');
});

test('before a search succeeds, the Time page says why it failed, with no section', async ({ page }, testInfo) => {
  await answerReport(page, { failure: 'find me: 401 Unauthorized' });
  await page.goto('/time');

  const line = page.getByRole('main').getByText(/^The last search failed/);
  await expect(line).toHaveText('The last search failed: find me: 401 Unauthorized.');
  await expect(line).toHaveCSS('padding-bottom', testInfo.project.name === 'phone' ? '48px' : '64px');
  await expect(heads(page)).toHaveCount(0);
});

for (const { name, status, says } of [
  { name: "Jira isn't set up", status: 404, says: "Jira isn't set up." },
  { name: "the report couldn't load", status: 502, says: "Couldn't load the time report." },
]) {
  test(`the Time page says when ${name}, with no section`, async ({ page }) => {
    await answerReport(page, status);
    await page.goto('/time');

    await expect(page.getByRole('main').getByText(says)).toBeVisible();
    await expect(heads(page)).toHaveCount(0);
  });
}

test('a long failure wraps on a narrow screen', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 640 });
  const failure = `search my worklogs: Get "https://api.atlassian.com/ex/jira/${'z'.repeat(200)}": EOF`;
  await answerReport(page, { ...report, failure });
  await page.goto('/time');

  await expect(page.getByRole('main').getByText(/^The last search failed/)).toContainText(failure);
  await expect(days(page)).toHaveCount(7);
  const scroll = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(scroll).toBe(0);
});
