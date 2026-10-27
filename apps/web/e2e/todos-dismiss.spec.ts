import type { Page, Route } from '@playwright/test';

import { home, tickets, todos } from './demo';
import { expect, test } from './fixtures';
import { group, heads, noteFirstHeads } from './groups';
import { answerDismissals, answerTickets } from './jira';

type Answer = Parameters<Route['fulfill']>[0];

/** The Todos page's Jira inbox. */
function inbox(page: Page) {
  return group(page, 'Jira inbox');
}

/** The inbox row of the ticket. */
function row(page: Page, key: string) {
  return inbox(page).getByRole('listitem').filter({ hasText: key });
}

/** The inbox row's Dismiss button for the ticket. */
function dismiss(page: Page, key: string) {
  return page.getByRole('button', { name: `Dismiss ${key}`, exact: true });
}

/** Answers each dismissal with the next answer, or as the service does a new one without. Gives the requests sent. */
async function serveDismissals(page: Page, ...answers: (Answer | Promise<Answer>)[]) {
  const sent: string[] = [];
  await page.route('/api/todos/dismissals/*', async (route) => {
    sent.push(`${route.request().method()} ${new URL(route.request().url()).pathname}`);
    await route.fulfill((await answers.shift()) ?? { status: 201 });
  });
  return sent;
}

/** The heads of the page without the inbox. */
const noInbox = ['High priority', 'Normal priority', 'Low priority', 'Jira tickets'];

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos } }));
  await answerTickets(page, tickets);
});

test('Dismiss takes the ticket out of the inbox, and leaves it in the Jira section', async ({ page }) => {
  await answerDismissals(page);
  const sent = await serveDismissals(page);
  await page.goto('/todos');
  await expect(inbox(page).getByText('3 tickets')).toBeVisible();

  await dismiss(page, 'WEB-9').click();
  await expect(inbox(page).getByRole('listitem')).toHaveText([/^WEB-12 /, /^OPS-31 /]);
  await expect(inbox(page).getByText('2 tickets')).toBeVisible();
  await expect(group(page, 'Jira tickets').getByRole('link', { name: /^WEB-9 / })).toBeVisible();
  await expect(page.getByRole('checkbox')).toHaveCount(todos.length);
  expect(sent).toEqual(['PUT /api/todos/dismissals/WEB-9']);
});

test('a ticket dismissed already, which the service answers with a 204, goes too', async ({ page }) => {
  await answerDismissals(page);
  await serveDismissals(page, { status: 204 });
  await page.goto('/todos');

  await dismiss(page, 'WEB-12').click();
  await expect(inbox(page).getByRole('listitem')).toHaveText([/^WEB-9 /, /^OPS-31 /]);
});

test('while a dismiss runs, its row says so, and neither of its buttons sends anything more', async ({ page }) => {
  await answerDismissals(page);
  const answer = Promise.withResolvers<Answer>();
  const sent = await serveDismissals(page, answer.promise);
  await page.goto('/todos');

  await dismiss(page, 'WEB-12').click();
  await expect(row(page, 'WEB-12').getByRole('status')).toHaveText('Dismissing…');
  for (const name of ['Add WEB-12', 'Dismiss WEB-12']) {
    await expect(row(page, 'WEB-12').getByRole('button', { name })).toHaveAttribute('aria-disabled', 'true');
  }
  await expect(dismiss(page, 'WEB-9')).toHaveAttribute('aria-disabled', 'false');
  await dismiss(page, 'WEB-12').dispatchEvent('click');
  answer.resolve({ status: 201 });

  await expect(row(page, 'WEB-12')).toHaveCount(0);
  expect(sent).toHaveLength(1);
});

test('a failed dismiss says so and keeps its row, and the next try dismisses', async ({ page }) => {
  await answerDismissals(page);
  const sent = await serveDismissals(page, { status: 500 });
  await page.goto('/todos');

  await dismiss(page, 'WEB-12').click();
  await expect(row(page, 'WEB-12').getByRole('status')).toHaveText('Dismiss failed');
  await expect(row(page, 'WEB-9').getByRole('status')).toHaveText('');

  await dismiss(page, 'WEB-12').click();
  await expect(row(page, 'WEB-12')).toHaveCount(0);
  expect(sent).toHaveLength(2);
});

test("Dismiss gives the focus to the next row's Dismiss, else the previous row's, else the Add field", async ({
  page,
}) => {
  await answerDismissals(page);
  await serveDismissals(page);
  await page.goto('/todos');

  await dismiss(page, 'WEB-9').press('Enter');
  await expect(dismiss(page, 'OPS-31')).toBeFocused();
  await dismiss(page, 'OPS-31').press('Enter');
  await expect(dismiss(page, 'WEB-12')).toBeFocused();
  await dismiss(page, 'WEB-12').press('Enter');
  await expect(page.getByRole('textbox', { name: 'New todo' })).toBeFocused();
  await expect(heads(page)).toHaveText(noInbox);
});

test('the tickets I dismissed before stay out of the inbox, and with all of them dismissed it goes', async ({
  page,
}) => {
  await answerDismissals(page, ['WEB-9', 'ABC-1']);
  await page.goto('/todos');
  await expect(inbox(page).getByRole('listitem')).toHaveText([/^WEB-12 /, /^OPS-31 /]);

  await answerDismissals(page, ['OPS-31', 'WEB-12', 'WEB-9']);
  await page.goto('/todos');
  await expect(page.getByRole('checkbox').first()).toBeVisible();
  await expect(heads(page)).toHaveText(noInbox);
});

test('when the dismissed tickets could not load, the page shows the todos and the tickets, but no inbox', async ({
  page,
}) => {
  await answerDismissals(page, 500);
  await page.goto('/todos');

  await expect(page.getByRole('checkbox').first()).toBeVisible();
  await expect(heads(page)).toHaveText(noInbox);
});

test('the todos wait for the dismissed tickets, so the inbox above them never pushes them down', async ({ page }) => {
  const held = Promise.withResolvers<void>();
  await page.route('/api/todos/dismissals', async (route) => {
    await held.promise;
    await route.fulfill({ json: { dismissals: ['WEB-12'] } });
  });
  await noteFirstHeads(page);
  const answered = [page.waitForResponse('/api/todos'), page.waitForResponse('/api/todos/tickets')];
  await page.goto('/todos');
  await Promise.all(answered);
  await expect(page.getByRole('heading', { level: 1, name: 'Todos' })).toBeVisible();
  held.resolve();

  await expect(inbox(page).getByRole('listitem')).toHaveText([/^WEB-9 /, /^OPS-31 /]);
  await expect(page.locator('html')).toHaveAttribute('data-heads', JSON.stringify(['Jira inbox', ...noInbox]));
});
