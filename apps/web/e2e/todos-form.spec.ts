import type { Page, Route } from '@playwright/test';

import type { Todo } from '../src/todos/api';
import { home, todos } from './demo';
import { expect, test } from './fixtures';

type Answer = Parameters<Route['fulfill']>[0];

/** Answers GET /api/todos with the list and each POST with add's answer; gives the bodies posted, in order. */
async function serveTodos(page: Page, list: Todo[], add: (title: string) => Answer | Promise<Answer>) {
  const posted: unknown[] = [];
  await page.route('/api/todos', async (route) => {
    if (route.request().method() !== 'POST') {
      await route.fulfill({ json: { todos: list } });
      return;
    }
    const body = route.request().postDataJSON() as { title: string };
    posted.push(body);
    await route.fulfill(await add(body.title));
  });
  return posted;
}

function added(title: string): Answer {
  return { status: 201, json: { id: 4, title: title.trim() } };
}

const titles = todos.map((todo) => todo.title);

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test('Enter adds the todo on top of the list', async ({ page }) => {
  const posted = await serveTodos(page, todos, added);
  await page.goto('/todos');
  const field = page.getByRole('textbox', { name: 'New todo' });

  await field.fill('  Pick up the dry cleaning ');
  await field.press('Enter');

  await expect(page.getByText('4 todos on the list.')).toBeVisible();
  await expect(page.getByRole('main').getByRole('listitem')).toHaveText(['Pick up the dry cleaning', ...titles]);
  await expect(page.getByRole('main').getByRole('status')).toHaveText('Added');
  await expect(field).toHaveValue('');
  await expect(field).toBeFocused();
  expect(posted).toEqual([{ title: '  Pick up the dry cleaning ' }]);
});

test('the Add button adds the first todo', async ({ page }) => {
  await serveTodos(page, [], added);
  await page.goto('/todos');
  await expect(page.getByText('No todos on the list.')).toBeVisible();
  await expect(page.getByRole('main').getByRole('list')).toHaveCount(0);

  await page.getByRole('textbox', { name: 'New todo' }).fill('Pick up the dry cleaning');
  await page.getByRole('button', { name: 'Add' }).click();

  await expect(page.getByText('1 todo on the list.')).toBeVisible();
  await expect(page.getByRole('main').getByRole('listitem')).toHaveText(['Pick up the dry cleaning']);
});

test('a refused title stays in the field, with the reason', async ({ page }) => {
  const title = `Read ${'x'.repeat(500)}`;
  await serveTodos(page, todos, () => ({ status: 400, body: 'invalid title: over 500 characters\n' }));
  await page.goto('/todos');
  const field = page.getByRole('textbox', { name: 'New todo' });

  await field.fill(title);
  await field.press('Enter');

  await expect(page.getByRole('main').getByRole('status')).toHaveText('invalid title: over 500 characters');
  await expect(field).toHaveValue(title);
  await expect(page.getByRole('main').getByRole('listitem')).toHaveText(titles);
});

test('a failed add keeps the text', async ({ page }) => {
  await serveTodos(page, todos, () => ({ status: 500, body: 'Internal Server Error\n' }));
  await page.goto('/todos');
  const field = page.getByRole('textbox', { name: 'New todo' });

  await field.fill('Pick up the dry cleaning');
  await field.press('Enter');

  await expect(page.getByRole('main').getByRole('status')).toHaveText('Add failed');
  await expect(field).toHaveValue('Pick up the dry cleaning');
  await expect(page.getByText('3 todos on the list.')).toBeVisible();
});

test('the form adds one todo at a time', async ({ page }) => {
  const answer = Promise.withResolvers<void>();
  const posted = await serveTodos(page, todos, async (title) => {
    await answer.promise;
    return added(title);
  });
  await page.goto('/todos');
  const field = page.getByRole('textbox', { name: 'New todo' });
  const button = page.getByRole('button', { name: 'Add' });

  await field.fill('Pick up the dry cleaning');
  await field.press('Enter');
  await expect(page.getByRole('main').getByRole('status')).toHaveText('Adding…');
  await expect(field).not.toBeEditable();
  await expect(button).toHaveAttribute('aria-disabled', 'true');
  await field.press('Enter');
  // A click on an aria-disabled button still submits the form, but Playwright's click() waits for
  // the button to be enabled, so the click is dispatched, as element.click() does.
  await button.dispatchEvent('click');
  answer.resolve();

  await expect(page.getByRole('main').getByRole('status')).toHaveText('Added');
  await expect(field).toBeEditable();
  await expect(button).toHaveAttribute('aria-disabled', 'false');
  expect(posted).toHaveLength(1);
});

test('a blank field adds nothing', async ({ page }) => {
  const posted = await serveTodos(page, todos, added);
  await page.goto('/todos');
  const field = page.getByRole('textbox', { name: 'New todo' });

  await field.fill('   ');
  await field.press('Enter');
  await page.getByRole('button', { name: 'Add' }).click();
  await expect(page.getByRole('main').getByRole('status')).toHaveText('');
  // The add that follows is the first POST, so the blank ones sent nothing.
  await field.fill('Pick up the dry cleaning');
  await field.press('Enter');

  await expect(page.getByRole('main').getByRole('status')).toHaveText('Added');
  expect(posted).toEqual([{ title: 'Pick up the dry cleaning' }]);
});
