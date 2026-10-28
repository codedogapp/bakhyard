import type { Page, Route } from '@playwright/test';

import type { Todo } from '../src/todos/api';
import { chipLooks } from './chips';
import { home, tickets, todos } from './demo';
import { expect, test } from './fixtures';
import { group, heads, noteFirstHeads } from './groups';
import { answerDismissals, answerTickets, aTodo } from './jira';

type Answer = Parameters<Route['fulfill']>[0];

/** The Todos page's Jira inbox. */
function inbox(page: Page) {
  return group(page, 'Jira inbox');
}

/** The inbox row of the ticket. */
function row(page: Page, key: string) {
  return inbox(page).getByRole('listitem').filter({ hasText: key });
}

/** The inbox row's Add button for the ticket. */
function add(page: Page, key: string) {
  return page.getByRole('button', { name: `Add ${key}`, exact: true });
}

/** The labels of a priority group's todos, in order: the title, then the ticket's key and status. */
function labels(page: Page, head: string) {
  return group(page, head).locator('label');
}

/**
 * Answers GET /api/todos with the list, and each POST with the next answer, or as the service does without one: the
 * todo it names, with the next ID. Gives the POSTs' bodies.
 */
async function serveAdds(page: Page, list: Todo[], ...answers: (Answer | Promise<Answer | undefined>)[]) {
  const sent: string[] = [];
  await page.route('/api/todos', async (route) => {
    const request = route.request();
    if (request.method() === 'GET') {
      return route.fulfill({ json: { todos: list } });
    }
    sent.push(request.postData() ?? '');
    const created = { id: 10 + sent.length, ...request.postDataJSON(), done: false, priority: 'normal' };
    return route.fulfill((await answers.shift()) ?? { status: 201, json: created });
  });
  return sent;
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await answerTickets(page, tickets);
  await answerDismissals(page);
});

test('the Todos page lists my open tickets that no todo names in an inbox before the groups', async ({ page }) => {
  const shipped: Todo = { id: 4, title: 'Ship the date picker fix', done: true, priority: 'high', ticket: 'WEB-9' };
  await serveAdds(page, [shipped, ...todos]);
  await page.goto('/todos');

  await expect(heads(page)).toHaveText([
    'Jira inbox',
    'High priority',
    'Normal priority',
    'Low priority',
    'Jira tickets',
  ]);
  await expect(inbox(page).getByText('2 tickets')).toBeVisible();
  await expect(inbox(page).getByRole('listitem')).toHaveText([
    'WEB-12 Show the order history on the account page In Review Add Dismiss',
    'OPS-31 Rotate the staging certificates To Do Add Dismiss',
  ]);
  expect(await chipLooks(inbox(page).locator('.chip'))).toEqual(['filled', 'outlined']);
  await expect(add(page, 'WEB-12')).toHaveText('Add');
  await expect(add(page, 'OPS-31')).toBeVisible();
  await expect(inbox(page).getByRole('link')).toHaveCount(0);
});

test("Add makes the ticket a todo on top of Normal, and gives the focus to the next row's Add", async ({ page }) => {
  const sent = await serveAdds(page, todos);
  await page.goto('/todos');

  await add(page, 'WEB-9').press('Enter');
  await expect(add(page, 'OPS-31')).toBeFocused();
  await expect(inbox(page).getByRole('listitem')).toHaveCount(2);
  await expect(labels(page, 'Normal priority').first()).toHaveText('Fix the date picker in Safari WEB-9 In Progress');
  await expect(group(page, 'Jira tickets').getByRole('link')).toHaveCount(3);

  // The last row gives it to the row before; the inbox's last row, with the section, to the Add field.
  await add(page, 'OPS-31').press('Enter');
  await expect(add(page, 'WEB-12')).toBeFocused();
  await add(page, 'WEB-12').press('Enter');
  await expect(page.getByRole('textbox', { name: 'New todo' })).toBeFocused();

  await expect(heads(page)).toHaveText(['High priority', 'Normal priority', 'Low priority', 'Jira tickets']);
  await expect(labels(page, 'Normal priority')).toHaveText([
    'Show the order history on the account page WEB-12 In Review',
    'Rotate the staging certificates OPS-31 To Do',
    'Fix the date picker in Safari WEB-9 In Progress',
    'Book the car in for its winter tyres',
  ]);
  expect(sent).toEqual([
    '{"title":"Fix the date picker in Safari","ticket":"WEB-9"}',
    '{"title":"Rotate the staging certificates","ticket":"OPS-31"}',
    '{"title":"Show the order history on the account page","ticket":"WEB-12"}',
  ]);
});

test("a refused add shows the service's reason on its row, a failed one says so, and the next try adds", async ({
  page,
}) => {
  const refusal = { status: 400, body: 'invalid title: over 500 characters\n' };
  const sent = await serveAdds(page, todos, refusal, { status: 500 });
  await page.goto('/todos');

  await add(page, 'WEB-12').click();
  await expect(row(page, 'WEB-12').getByRole('status')).toHaveText('invalid title: over 500 characters');
  await add(page, 'WEB-12').click();
  await expect(row(page, 'WEB-12').getByRole('status')).toHaveText('Add failed');
  await expect(row(page, 'WEB-9').getByRole('status')).toHaveText('');

  await add(page, 'WEB-12').click();
  await expect(row(page, 'WEB-12')).toHaveCount(0);
  expect(sent).toHaveLength(3);
});

test('while an add runs, its row says so and its Add sends nothing more', async ({ page }) => {
  const answer = Promise.withResolvers<undefined>();
  const sent = await serveAdds(page, todos, answer.promise);
  await page.goto('/todos');

  await add(page, 'WEB-12').click();
  await expect(row(page, 'WEB-12').getByRole('status')).toHaveText('Adding…');
  await expect(add(page, 'WEB-12')).toHaveAttribute('aria-disabled', 'true');
  await expect(add(page, 'WEB-9')).toHaveAttribute('aria-disabled', 'false');
  await add(page, 'WEB-12').dispatchEvent('click');
  answer.resolve(undefined);

  await expect(row(page, 'WEB-12')).toHaveCount(0);
  expect(sent).toHaveLength(1);
});

test('deleting a todo or taking its ticket off puts the ticket back in the inbox, in its place', async ({ page }) => {
  const shipping: Todo = { id: 5, title: 'Ship the date picker fix', done: false, priority: 'normal', ticket: 'WEB-9' };
  const rotation: Todo = { id: 4, title: 'Plan the rotation', done: false, priority: 'normal', ticket: 'OPS-31' };
  await serveAdds(page, [shipping, rotation]);
  await page.route(aTodo, (route) =>
    route.request().method() === 'DELETE'
      ? route.fulfill({ status: 204 })
      : route.fulfill({ json: { ...rotation, ticket: undefined } })
  );
  await page.goto('/todos');
  await expect(inbox(page).getByRole('listitem')).toHaveText([/^WEB-12 /]);

  await page.getByRole('button', { name: 'Edit Ship the date picker fix' }).click();
  await page.getByRole('main').getByRole('listitem').getByRole('button', { name: 'Delete' }).click();
  await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click();
  await expect(inbox(page).getByRole('listitem')).toHaveText([/^WEB-12 /, /^WEB-9 /]);

  await page.getByRole('button', { name: 'Edit Plan the rotation' }).click();
  await page.getByRole('combobox', { name: 'Ticket' }).selectOption('');
  await page.getByRole('button', { name: 'Save' }).click();
  await expect(inbox(page).getByRole('listitem')).toHaveText([/^WEB-12 /, /^WEB-9 /, /^OPS-31 /]);
});

test('the todos wait for my open tickets, so the inbox above them never pushes them down', async ({ page }) => {
  const held = Promise.withResolvers<void>();
  await page.route('/api/todos/tickets', async (route) => {
    await held.promise;
    await route.fulfill({ json: tickets });
  });
  await serveAdds(page, todos);
  await noteFirstHeads(page);
  const listed = page.waitForResponse('/api/todos');
  await page.goto('/todos');
  await listed;
  await expect(page.getByRole('heading', { level: 1, name: 'Todos' })).toBeVisible();
  held.resolve();

  await expect(page.getByRole('checkbox').first()).toBeVisible();
  const all = ['Jira inbox', 'High priority', 'Normal priority', 'Low priority', 'Jira tickets'];
  await expect(page.locator('html')).toHaveAttribute('data-heads', JSON.stringify(all));
});
