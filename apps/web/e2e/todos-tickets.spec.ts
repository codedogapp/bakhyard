import type { Page } from '@playwright/test';

import { home, tickets, todos } from './demo';
import { expect, test } from './fixtures';
import { group, heads } from './groups';
import { answerTickets, chipLooks } from './jira';

/** The Todos page's Jira section. */
function jira(page: Page) {
  return group(page, 'Jira tickets');
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos } }));
});

test('the Todos page lists my open Jira tickets after the todo groups', async ({ page }) => {
  await answerTickets(page, tickets);
  await page.goto('/todos');

  await expect(heads(page)).toHaveText(['High priority', 'Normal priority', 'Low priority', 'Jira tickets']);
  await expect(jira(page).getByText('3 tickets')).toBeVisible();
  const rows = jira(page).getByRole('link');
  await expect(rows).toHaveText([
    'WEB-12 Show the order history on the account page In Review',
    'WEB-9 Fix the date picker in Safari In Progress',
    'OPS-31 Rotate the staging certificates To Do',
  ]);
  for (const [index, ticket] of (tickets.tickets ?? []).entries()) {
    await expect(rows.nth(index)).toHaveAttribute('href', ticket.url);
  }
  // The In Progress category's chips are filled with the ink; the others keep only its outline, and its ink for the text.
  expect(await chipLooks(jira(page).locator('.chip'))).toEqual(['filled', 'filled', 'outlined']);
});

test("a ticket's row takes me to the ticket in Jira", async ({ page }) => {
  await answerTickets(page, tickets);
  await page.route('https://jira.example.com/**', (route) =>
    route.fulfill({ contentType: 'text/html', body: '<title>WEB-12</title>' })
  );
  await page.goto('/todos');
  await jira(page)
    .getByRole('link', { name: /^WEB-12 / })
    .click();

  await expect(page).toHaveURL('https://jira.example.com/browse/WEB-12');
  await expect(page).toHaveTitle('WEB-12');
});

test('the Todos page loads the tickets again each time it shows', async ({ page }) => {
  let snapshot = tickets;
  await page.route('/api/todos/tickets', (route) => route.fulfill({ json: snapshot }));
  await page.goto('/todos');
  await expect(jira(page).getByText('3 tickets')).toBeVisible();

  await page.getByRole('link', { name: 'Home', exact: true }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Good morning, Alex.' })).toBeVisible();
  // The todos service finds a ticket gone while the page is away.
  snapshot = { ...tickets, tickets: tickets.tickets?.slice(1) };
  await page.getByRole('link', { name: 'Todos', exact: true }).click();

  await expect(jira(page).getByText('2 tickets')).toBeVisible();
  await expect(jira(page).getByRole('link')).toHaveCount(2);
});

test('the Todos page says why the last search failed, and when its tickets are from', async ({ page }) => {
  await answerTickets(page, { ...tickets, failure: 'search Jira: 502 Bad Gateway' });
  await page.goto('/todos');

  const line = jira(page).getByText(/^The last search failed/);
  await expect(line).toHaveText(
    'The last search failed: search Jira: 502 Bad Gateway. These tickets are from 21 October at 09:38.'
  );
  await expect(line.locator('time')).toHaveAttribute('datetime', '2026-10-21T00:38:00Z');
  await expect(jira(page).getByRole('link')).toHaveCount(3);
});

test('before a search succeeds, the Todos page says why it failed, with no list', async ({ page }) => {
  await answerTickets(page, { failure: 'search Jira: 401 Unauthorized' });
  await page.goto('/todos');

  await expect(jira(page).getByText(/^The last search failed/)).toHaveText(
    'The last search failed: search Jira: 401 Unauthorized.'
  );
  await expect(jira(page).getByRole('list')).toHaveCount(0);
  await expect(jira(page).getByText(/^\d+ tickets?$/)).toHaveCount(0);
});

test('the Todos page says when no ticket is open', async ({ page }) => {
  await answerTickets(page, { tickets: [], searched: '2026-10-21T00:38:00Z' });
  await page.goto('/todos');

  await expect(jira(page).getByText('No open tickets.')).toBeVisible();
  await expect(jira(page).getByText(/^\d+ tickets?$/)).toHaveCount(0);
});

test('the Todos page says when the tickets could not load', async ({ page }) => {
  await answerTickets(page, 500);
  await page.goto('/todos');

  await expect(jira(page).getByText("Couldn't load the tickets.")).toBeVisible();
  await expect(jira(page).getByRole('list')).toHaveCount(0);
});

test('the tickets follow the line that says the todos could not load', async ({ page }) => {
  await page.route('/api/todos', (route) => route.fulfill({ status: 500 }));
  await answerTickets(page, tickets);
  await page.goto('/todos');

  const head = page.getByRole('heading', { level: 2, name: 'Jira tickets' });
  await expect(head).toBeVisible();
  const [above, below] = [await page.getByText("Couldn't load the todos.").boundingBox(), await head.boundingBox()];
  expect(above && below && above.y + above.height <= below.y).toBe(true);
});

test('a long key, summary, status or failure wraps on a narrow screen', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 640 });
  const ticket = {
    key: 'ABCDEFGHIJ-1234567',
    summary: `Read ${'x'.repeat(200)}`,
    status: 'Waiting for the customer to approve the change',
    category: 'to-do',
    url: 'https://jira.example.com/browse/ABCDEFGHIJ-1234567',
  } as const;
  const failure = `search Jira: Get "https://jira.example.com/rest/api/3/search/jql?${'z'.repeat(200)}": EOF`;
  await answerTickets(page, { tickets: [ticket], searched: '2026-10-21T00:38:00Z', failure });
  await page.goto('/todos');

  await expect(jira(page).getByRole('link')).toHaveText(`${ticket.key} ${ticket.summary} ${ticket.status}`);
  const scroll = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(scroll).toBe(0);
  const [key, chip] = [
    await jira(page).getByText(ticket.key).boundingBox(),
    await jira(page).locator('.chip').boundingBox(),
  ];
  expect(key && chip && key.x + key.width <= chip.x).toBe(true);
  expect(chip?.width).toBeLessThanOrEqual(160);
});
