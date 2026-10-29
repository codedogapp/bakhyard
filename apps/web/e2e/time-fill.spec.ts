import type { Page, Route } from '@playwright/test';

import type { Report } from '../src/time/report';
import { home, report, tickets } from './demo';
import { expect, test } from './fixtures';
import { answerTickets } from './jira';

type Answer = Parameters<Route['fulfill']>[0];

/** Answers the Time page's calls for the report with each in turn, the last again; gives the number of calls. */
async function answerReports(page: Page, ...reports: Report[]) {
  const calls = { count: 0 };
  await page.route('/api/time/report', (route) => {
    calls.count += 1;
    return route.fulfill({ json: reports[Math.min(calls.count, reports.length) - 1] });
  });
  return calls;
}

/** Answers each worklog the form adds with the answer; gives the bodies sent, in order. */
async function serveLogs(page: Page, answer: () => Answer | Promise<Answer>) {
  const sent: unknown[] = [];
  await page.route('/api/time/worklogs', async (route) => {
    sent.push(route.request().postDataJSON());
    await route.fulfill(await answer());
  });
  return sent;
}

const added: Answer = { status: 201 };

// The demo report, once Friday's 7h 30m are on WEB-9.
const filled: Report = {
  ...report,
  days: report.days?.map((day) => (day.date === '2026-10-16' ? { ...day, loggedHours: 7.5, missing: false } : day)),
  week: { loggedHours: 23.25, targetHours: 40 },
};

const select = (page: Page, name: string) => page.getByRole('main').getByRole('combobox', { name });
const logButton = (page: Page) => page.getByRole('main').getByRole('button', { name: 'Log', exact: true });
const dialog = (page: Page) => page.getByRole('alertdialog', { name: 'Log these hours?' });
const status = (page: Page) => page.getByRole('main').getByRole('status');

/** Picks WEB-9 and the minutes in the form, then asks to log them. */
async function ask(page: Page, minutes = '480') {
  await select(page, 'Ticket').selectOption('WEB-9');
  await select(page, 'Hours').selectOption(minutes);
  await logButton(page).click();
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await answerTickets(page, tickets);
});

test('the form offers the last 7 days, my open tickets and the hours, and starts on the missing Friday', async ({
  page,
}) => {
  await answerReports(page, report);
  await page.goto('/time');

  await expect(select(page, 'Day')).toHaveValue('2026-10-16');
  await expect(select(page, 'Day').getByRole('option')).toHaveText([
    'Thursday 15 Oct',
    'Friday 16 Oct',
    'Saturday 17 Oct',
    'Sunday 18 Oct',
    'Monday 19 Oct',
    'Tuesday 20 Oct',
    'Today',
  ]);
  await expect(select(page, 'Ticket')).toHaveValue('');
  await expect(select(page, 'Ticket').getByRole('option')).toHaveText([
    'Pick a ticket',
    'WEB-12 · Show the order history on the account page',
    'WEB-9 · Fix the date picker in Safari',
    'OPS-31 · Rotate the staging certificates',
  ]);
  await expect(select(page, 'Hours')).toHaveValue('480');
  await expect(select(page, 'Hours').getByRole('option')).toHaveCount(48);
  await expect(select(page, 'Hours').getByRole('option').first()).toHaveText('0h 15m');
  await expect(select(page, 'Hours').getByRole('option').last()).toHaveText('12h');
  await expect(status(page)).toHaveText('');
});

test('Log with no ticket picked opens nothing, and the browser asks for one', async ({ page }) => {
  await answerReports(page, report);
  const sent = await serveLogs(page, () => added);
  await page.goto('/time');

  await logButton(page).click();

  await expect(dialog(page)).toBeHidden();
  expect(await select(page, 'Ticket').evaluate((element: HTMLSelectElement) => element.validity.valueMissing)).toBe(
    true
  );
  expect(sent).toEqual([]);
});

test('Log asks in a modal dialog, with the focus on Cancel; Cancel and Escape give it back to Log', async ({
  page,
}) => {
  await answerReports(page, report);
  const sent = await serveLogs(page, () => added);
  await page.goto('/time');

  // A click, which in WebKit focuses no button, so the browser has no focus to give back.
  await ask(page);
  await expect(dialog(page)).toHaveAccessibleDescription(
    '8h on WEB-9 · Fix the date picker in Safari, from 09:00 on Friday 16 Oct.'
  );
  expect(await dialog(page).evaluate((element) => element.matches(':modal'))).toBe(true);
  await expect(dialog(page).getByRole('button', { name: 'Cancel' })).toBeFocused();
  await dialog(page).getByRole('button', { name: 'Cancel' }).click();
  await expect(dialog(page)).toBeHidden();
  await expect(logButton(page)).toBeFocused();
  await logButton(page).press('Enter');
  await expect(dialog(page)).toBeVisible();
  await page.keyboard.press('Escape');

  await expect(dialog(page)).toBeHidden();
  await expect(logButton(page)).toBeFocused();
  expect(sent).toEqual([]);
});

test("the dialog's Log adds the worklog and shows the hours, and the form keeps its values", async ({ page }) => {
  const reports = await answerReports(page, report, filled);
  const sent = await serveLogs(page, () => added);
  await page.goto('/time');

  await ask(page, '450');
  await dialog(page).getByRole('button', { name: 'Log' }).click();

  await expect(status(page)).toHaveText('Logged');
  await expect(page.getByRole('main').getByRole('listitem').nth(1)).toHaveText('16 Oct Friday 7h 30m');
  await expect(page.getByText('23h 15m / 40h')).toBeVisible();
  await expect(logButton(page)).toBeFocused();
  // The day logged stays picked, though Friday misses nothing now.
  await expect(select(page, 'Day')).toHaveValue('2026-10-16');
  await expect(select(page, 'Ticket')).toHaveValue('WEB-9');
  await expect(select(page, 'Hours')).toHaveValue('450');
  // The form submits nothing itself: no "?" on the address, no step in the history.
  await expect(page).toHaveURL('/time');
  expect(sent).toEqual([{ ticket: 'WEB-9', date: '2026-10-16', minutes: 450 }]);
  expect(reports.count).toBe(2);
});

for (const { name, answer, says } of [
  {
    name: 'refused',
    answer: { status: 400, body: 'invalid date: not one of the last 7 days\n' },
    says: 'invalid date: not one of the last 7 days',
  },
  { name: 'failed', answer: { status: 502, body: 'Bad Gateway\n' }, says: 'Log failed' },
]) {
  test(`a ${name} log says so, keeps the hours on the page and the form's values`, async ({ page }) => {
    const reports = await answerReports(page, report);
    const sent = await serveLogs(page, () => answer);
    await page.goto('/time');

    await ask(page);
    await dialog(page).getByRole('button', { name: 'Log' }).click();

    await expect(status(page)).toHaveText(says);
    await expect(page.getByRole('main').getByRole('listitem').nth(1)).toHaveText('16 Oct Friday Missing 0h');
    await expect(select(page, 'Ticket')).toHaveValue('WEB-9');
    expect(sent).toHaveLength(1);
    expect(reports.count).toBe(1);
  });
}

test('while a log runs, the form says so and Log opens nothing, but the fields stay editable', async ({ page }) => {
  await answerReports(page, report, filled);
  const answer = Promise.withResolvers<Answer>();
  const sent = await serveLogs(page, () => answer.promise);
  await page.goto('/time');

  await ask(page);
  await dialog(page).getByRole('button', { name: 'Log' }).click();
  await expect(status(page)).toHaveText('Logging…');
  await expect(logButton(page)).toHaveAttribute('aria-disabled', 'true');
  await logButton(page).dispatchEvent('click');
  await expect(dialog(page)).toBeHidden();
  await select(page, 'Hours').selectOption('60');
  answer.resolve(added);

  await expect(status(page)).toHaveText('Logged');
  await expect(logButton(page)).toHaveAttribute('aria-disabled', 'false');
  await expect(select(page, 'Hours')).toHaveValue('60');
  expect(sent).toEqual([{ ticket: 'WEB-9', date: '2026-10-16', minutes: 480 }]);
});

for (const { name, answer, says } of [
  { name: 'no ticket is open', answer: { ...tickets, tickets: [] }, says: 'No open Jira tickets to log hours on.' },
  { name: "my tickets couldn't load", answer: 502, says: "Couldn't load your Jira tickets." },
]) {
  test(`when ${name}, a line takes the form's place`, async ({ page }) => {
    await answerReports(page, report);
    await answerTickets(page, answer);
    await page.goto('/time');

    await expect(page.getByRole('main').getByText(says)).toBeVisible();
    await expect(select(page, 'Day')).toBeHidden();
    await expect(page.getByRole('main').getByRole('listitem')).toHaveCount(7);
  });
}

test('the form wraps on a 320 px screen, and the dialog fits it', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 800 });
  await answerReports(page, report);
  await page.goto('/time');

  await ask(page);
  await expect(dialog(page)).toBeVisible();
  expect(await dialog(page).boundingBox()).toMatchObject({ x: 20, width: 280 });
  expect(await dialog(page).evaluate((element) => element.scrollWidth - element.clientWidth)).toBe(0);
  const scroll = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(scroll).toBe(0);
});
