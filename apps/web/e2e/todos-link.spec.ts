import type { Page, Route } from '@playwright/test';

import type { Todo } from '../src/todos/api';
import { home, tickets } from './demo';
import { expect, test } from './fixtures';
import { answerDismissals, answerTickets, aTodo, jiraOff } from './jira';

type Answer = Parameters<Route['fulfill']>[0];

/** Made-up todos: one names an open ticket of the demo's snapshot, one a ticket that isn't open, one none. */
const shipping: Todo = { id: 3, title: 'Ship the date picker fix', done: false, priority: 'normal', ticket: 'WEB-9' };
const archive: Todo = { id: 2, title: 'Archive the old export', done: false, priority: 'normal', ticket: 'WEB-4' };
const milk: Todo = { id: 1, title: 'Buy milk', done: false, priority: 'normal' };
const list = [shipping, archive, milk];

/** My open tickets, as the editor offers them after "No ticket". */
const mine = [
  'WEB-12 · Show the order history on the account page',
  'WEB-9 · Fix the date picker in Safari',
  'OPS-31 · Rotate the staging certificates',
];

/** The service's answer to an edit: the todo with the edit's members, and no ticket once "" unlinked it. */
function edited(todo: Todo, edit: Partial<Todo>): Answer {
  const { ticket, ...saved } = { ...todo, ...edit };
  return { json: ticket ? { ...saved, ticket } : saved };
}

/** Answers GET /api/todos with the list, and each PATCH as the service does or with the answer; gives their bodies. */
async function serveEdits(page: Page, answer?: Promise<Answer>) {
  const sent: string[] = [];
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos: list } }));
  await page.route(aTodo, async (route) => {
    const request = route.request();
    sent.push(`${request.method()} ${new URL(request.url()).pathname} ${request.postData()}`);
    const todo = list.find(({ id }) => request.url().endsWith(`/${id}`));
    await route.fulfill(await (answer ?? edited(todo as Todo, request.postDataJSON())));
  });
  return sent;
}

const ticket = (page: Page) => page.getByRole('combobox', { name: 'Ticket' });
const edit = (page: Page, todo: Todo) => page.getByRole('button', { name: `Edit ${todo.title}`, exact: true });
const box = (page: Page, name: string) => page.getByRole('checkbox', { name, exact: true });

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await answerDismissals(page);
});

test('a todo with no ticket picks one of my open tickets, which its row then shows', async ({ page }) => {
  await answerTickets(page, tickets);
  const sent = await serveEdits(page);
  await page.goto('/todos');

  await edit(page, milk).click();
  await expect(ticket(page).locator('option')).toHaveText(['No ticket', ...mine]);
  await expect(ticket(page)).toHaveValue('');
  await ticket(page).selectOption('OPS-31');
  await page.getByRole('button', { name: 'Save' }).click();

  await expect(box(page, 'Buy milk OPS-31 To Do')).toBeVisible();
  await expect(edit(page, milk)).toBeFocused();
  expect(sent).toEqual(['PATCH /api/todos/1 {"ticket":"OPS-31"}']);
});

test("No ticket unlinks the todo's ticket, and its row drops it", async ({ page }) => {
  await answerTickets(page, tickets);
  const sent = await serveEdits(page);
  await page.goto('/todos');

  await edit(page, shipping).click();
  await expect(ticket(page).locator('option')).toHaveText(['No ticket', ...mine]);
  await expect(ticket(page)).toHaveValue('WEB-9');
  await ticket(page).selectOption({ label: 'No ticket' });
  await page.getByRole('button', { name: 'Save' }).click();

  await expect(box(page, shipping.title)).toBeVisible();
  expect(sent).toEqual(['PATCH /api/todos/3 {"ticket":""}']);
});

test("a ticket that isn't open shows by its key alone, and a new title goes without it", async ({ page }) => {
  await answerTickets(page, tickets);
  const sent = await serveEdits(page);
  await page.goto('/todos');

  await edit(page, archive).click();
  await expect(ticket(page).locator('option')).toHaveText(['No ticket', 'WEB-4', ...mine]);
  await expect(ticket(page)).toHaveValue('WEB-4');
  await page.getByRole('textbox', { name: 'Title' }).fill('Archive the old exports');
  await page.getByRole('button', { name: 'Save' }).click();

  await expect(box(page, 'Archive the old exports WEB-4')).toBeVisible();
  expect(sent).toEqual(['PATCH /api/todos/2 {"title":"Archive the old exports"}']);
});

test('without Jira, a todo with no ticket has none to pick, and one with a ticket can drop it', async ({ page }) => {
  await jiraOff(page);
  const sent = await serveEdits(page);
  await page.goto('/todos');

  await edit(page, milk).click();
  await expect(page.getByRole('textbox', { name: 'Title' })).toBeFocused();
  await expect(ticket(page)).toHaveCount(0);
  await edit(page, shipping).click();
  await expect(ticket(page).locator('option')).toHaveText(['No ticket', 'WEB-9']);
  await ticket(page).selectOption('');
  await page.getByRole('button', { name: 'Save' }).click();

  await expect(box(page, shipping.title)).toBeVisible();
  expect(sent).toEqual(['PATCH /api/todos/3 {"ticket":""}']);
});

test('Cancel and Escape drop the ticket picked, and one picked back closes with no request', async ({ page }) => {
  await answerTickets(page, tickets);
  const sent = await serveEdits(page);
  await page.goto('/todos');

  await edit(page, shipping).click();
  await ticket(page).selectOption('WEB-12');
  await page.getByRole('button', { name: 'Cancel' }).click();
  await edit(page, shipping).click();
  await expect(ticket(page)).toHaveValue('WEB-9');
  await ticket(page).selectOption('');
  await page.getByRole('textbox', { name: 'Title' }).press('Escape');
  await edit(page, shipping).click();
  await expect(ticket(page)).toHaveValue('WEB-9');
  await ticket(page).selectOption('OPS-31');
  await ticket(page).selectOption('WEB-9');
  await page.getByRole('button', { name: 'Save' }).click();

  await expect(edit(page, shipping)).toBeFocused();
  await expect(box(page, 'Ship the date picker fix WEB-9 In Progress')).toBeVisible();
  expect(sent).toEqual([]);
});

test('while a save runs, the ticket waits for it, disabled and muted as the buttons', async ({ page }) => {
  await answerTickets(page, tickets);
  const answer = Promise.withResolvers<Answer>();
  const sent = await serveEdits(page, answer.promise);
  await page.goto('/todos');
  const save = page.getByRole('button', { name: 'Save' });

  await edit(page, milk).click();
  await ticket(page).selectOption('WEB-12');
  const ink = await ticket(page).evaluate((select) => getComputedStyle(select).color);
  await save.click();
  await expect(save).toHaveAttribute('aria-disabled', 'true');
  const muted = await save.evaluate((button) => getComputedStyle(button).color);
  expect(muted).not.toBe(ink);
  await expect(ticket(page)).toBeDisabled();
  await expect(ticket(page)).toHaveCSS('color', muted);
  await expect(ticket(page)).toHaveCSS('opacity', '1');
  await expect(ticket(page)).toHaveCSS('cursor', 'default');
  await expect(page.locator('form svg')).toHaveCSS('stroke', muted);
  answer.resolve(edited(milk, { ticket: 'WEB-12' }));

  await expect(box(page, 'Buy milk WEB-12 In Review')).toBeVisible();
  expect(sent).toEqual(['PATCH /api/todos/1 {"ticket":"WEB-12"}']);
});
