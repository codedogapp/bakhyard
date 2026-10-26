import type { Page } from '@playwright/test';

import type { Todo } from '../src/todos/api';
import { home, todos } from './demo';
import { expect, test } from './fixtures';
import { group, heads } from './groups';

const [notes, car, passport] = todos;
const cleaning: Todo = { id: 4, title: 'Pick up the dry cleaning', done: false, priority: 'normal' };

/** Answers GET /api/todos with the list, each POST with the dry cleaning and each DELETE with a 204. */
async function serveTodos(page: Page, list: Todo[]) {
  await page.route('/api/todos', (route) =>
    route.request().method() === 'POST'
      ? route.fulfill({ status: 201, json: cleaning })
      : route.fulfill({ json: { todos: list } })
  );
  await page.route('/api/todos/*', (route) => route.fulfill({ status: 204 }));
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test('the Todos page shows a section per priority, High first, each with its count and its todos', async ({ page }) => {
  await serveTodos(page, [cleaning, ...todos]);
  await page.goto('/todos');

  await expect(page.getByText('4 todos on the list.')).toBeVisible();
  await expect(heads(page)).toHaveText(['High priority', 'Normal priority', 'Low priority']);
  await expect(heads(page).first().locator('em')).toHaveText('priority');
  for (const [head, count, rows] of [
    ['High priority', '1 todo', [notes]],
    ['Normal priority', '2 todos', [cleaning, car]],
    ['Low priority', '1 todo', [passport]],
  ] as const) {
    await expect(group(page, head).getByText(count, { exact: true })).toBeVisible();
    await expect(group(page, head).getByRole('listitem').locator('label')).toHaveText(rows.map((todo) => todo.title));
  }
});

test('a priority without todos has no section', async ({ page }) => {
  await serveTodos(page, [car, passport]);
  await page.goto('/todos');

  await expect(heads(page)).toHaveText(['Normal priority', 'Low priority']);
});

test('a new todo lands on top of the Normal group, which shows if it was not there', async ({ page }) => {
  await serveTodos(page, [notes]);
  await page.goto('/todos');
  await expect(heads(page)).toHaveText(['High priority']);

  await page.getByRole('textbox', { name: 'New todo' }).fill(cleaning.title);
  await page.getByRole('button', { name: 'Add' }).click();

  await expect(heads(page)).toHaveText(['High priority', 'Normal priority']);
  await expect(group(page, 'Normal priority').getByRole('listitem').locator('label')).toHaveText([cleaning.title]);
});

test("deleting a group's last todo takes its section away, and the focus moves on to the next group", async ({
  page,
}) => {
  await serveTodos(page, todos);
  await page.goto('/todos');

  await page.getByRole('button', { name: `Edit ${notes.title}` }).click();
  await page.getByRole('main').getByRole('listitem').getByRole('button', { name: 'Delete' }).click();
  await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click();

  await expect(heads(page)).toHaveText(['Normal priority', 'Low priority']);
  await expect(page.getByRole('button', { name: `Edit ${car.title}` })).toBeFocused();
});
