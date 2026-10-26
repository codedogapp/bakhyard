import type { Page, Route } from '@playwright/test';

import type { Todo } from '../src/todos/api';
import { home, todos } from './demo';
import { expect, test } from './fixtures';
import { group, heads } from './groups';

type Answer = Parameters<Route['fulfill']>[0];
type Edit = (todo: Todo, edit: Partial<Todo>) => Answer | Promise<Answer>;

const [notes, car, passport] = todos;

/** The service's answer to an edit: the todo with the edit's members, its title trimmed. */
const edited: Edit = (todo, edit) => {
  const saved = { ...todo, ...edit };
  return { json: { ...saved, title: saved.title.trim() } };
};

/** Answers GET /api/todos with the list and each PATCH with the edit's answer; gives the PATCHes sent, in order. */
async function serveEdits(page: Page, list: Todo[], answer: Edit = edited) {
  const sent: string[] = [];
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos: list } }));
  await page.route('/api/todos/*', async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    sent.push(`${request.method()} ${path} ${request.postData()}`);
    const todo = list.find(({ id }) => id === Number(path.split('/').pop()));
    await route.fulfill(await answer(todo as Todo, request.postDataJSON()));
  });
  return sent;
}

/** The open editor's radio for the priority. */
function radio(page: Page, name: string) {
  return page.getByRole('group', { name: 'Priority' }).getByRole('radio', { name, exact: true });
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test("a new priority moves the todo to its group, and the focus to the todo's Edit button there", async ({ page }) => {
  const sent = await serveEdits(page, [car, passport]);
  await page.goto('/todos');
  await expect(heads(page)).toHaveText(['Normal priority', 'Low priority']);

  await page.getByRole('button', { name: `Edit ${car.title}` }).click();
  await expect(radio(page, 'Normal')).toBeChecked();
  await radio(page, 'High').click();
  await page.getByRole('button', { name: 'Save' }).click();

  await expect(heads(page)).toHaveText(['High priority', 'Low priority']);
  await expect(group(page, 'High priority').getByRole('listitem').locator('label')).toHaveText([car.title]);
  await expect(group(page, 'High priority').getByText('1 todo', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: `Edit ${car.title}` })).toBeFocused();
  expect(sent).toEqual(['PATCH /api/todos/2 {"priority":"high"}']);
});

test('a new title and a priority picked with the arrow keys go in one request', async ({ page }) => {
  const sent = await serveEdits(page, todos);
  await page.goto('/todos');

  await page.getByRole('button', { name: `Edit ${car.title}` }).click();
  await page.getByRole('textbox', { name: 'Title' }).fill('Book the car in for its MOT');
  await radio(page, 'Normal').focus();
  await page.keyboard.press('ArrowDown');
  await expect(radio(page, 'Low')).toBeChecked();
  await expect(radio(page, 'Low')).toBeFocused();
  await page.getByRole('button', { name: 'Save' }).click();

  await expect(heads(page)).toHaveText(['High priority', 'Low priority']);
  await expect(group(page, 'Low priority').getByRole('listitem').locator('label')).toHaveText([
    'Book the car in for its MOT',
    passport.title,
  ]);
  await expect(page.getByRole('button', { name: 'Edit Book the car in for its MOT' })).toBeFocused();
  expect(sent).toEqual(['PATCH /api/todos/2 {"title":"Book the car in for its MOT","priority":"low"}']);
});

test('Tab stops once in the priority, on the one the todo has', async ({ page, browserName }) => {
  // Playwright's WebKit, like Safari unless "Press Tab to highlight each item" is on, moves Tab
  // only between text fields (plan §6.8).
  test.skip(browserName === 'webkit', 'WebKit skips radios on Tab');
  await serveEdits(page, todos);
  await page.goto('/todos');

  await page.getByRole('button', { name: `Edit ${car.title}` }).click();
  await page.keyboard.press('Tab');
  await expect(radio(page, 'Normal')).toBeFocused();
  await page.keyboard.press('Tab');
  await expect(page.getByRole('button', { name: 'Save' })).toBeFocused();
});

test('Cancel and Escape drop the priority picked, and one picked back closes with no request', async ({ page }) => {
  const sent = await serveEdits(page, todos);
  await page.goto('/todos');
  const edit = page.getByRole('button', { name: `Edit ${car.title}` });

  await edit.click();
  await radio(page, 'High').click();
  await page.getByRole('button', { name: 'Cancel' }).click();
  await edit.click();
  await expect(radio(page, 'Normal')).toBeChecked();
  await radio(page, 'Low').click();
  await page.getByRole('textbox', { name: 'Title' }).press('Escape');
  await edit.click();
  await expect(radio(page, 'Normal')).toBeChecked();
  await radio(page, 'High').click();
  await radio(page, 'Normal').click();
  await page.getByRole('button', { name: 'Save' }).click();

  await expect(edit).toBeFocused();
  await expect(heads(page)).toHaveText(['High priority', 'Normal priority', 'Low priority']);
  expect(sent).toEqual([]);
});

test('while a save runs, the priority waits for it, muted as the buttons', async ({ page }) => {
  const answer = Promise.withResolvers<Answer>();
  const sent = await serveEdits(page, todos, () => answer.promise);
  await page.goto('/todos');
  const save = page.getByRole('button', { name: 'Save' });

  await page.getByRole('button', { name: `Edit ${notes.title}` }).click();
  await radio(page, 'Low').click();
  await save.click();
  await expect(save).toHaveAttribute('aria-disabled', 'true');
  const muted = await save.evaluate((button) => getComputedStyle(button).color);
  for (const name of ['High', 'Normal', 'Low']) {
    await expect(radio(page, name)).toHaveAttribute('aria-disabled', 'true');
    await expect(radio(page, name)).toHaveCSS('border-top-color', muted);
    await expect(page.getByRole('group', { name: 'Priority' }).getByText(name, { exact: true })).toHaveCSS(
      'color',
      muted
    );
  }
  await expect(radio(page, 'Low')).toHaveCSS('background-color', muted);
  await radio(page, 'High').dispatchEvent('click');
  await radio(page, 'Low').focus();
  await page.keyboard.press('ArrowUp');
  await expect(radio(page, 'Low')).toBeChecked();
  answer.resolve(edited(notes, { priority: 'low' }));

  await expect(heads(page)).toHaveText(['Normal priority', 'Low priority']);
  await expect(group(page, 'Low priority').getByRole('listitem').locator('label')).toHaveText([
    notes.title,
    passport.title,
  ]);
  expect(sent).toEqual(['PATCH /api/todos/3 {"priority":"low"}']);
});
