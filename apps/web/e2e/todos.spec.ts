import { home, todos } from './demo';
import { expect, test } from './fixtures';

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test('the Todos page lists the todos, newest first', async ({ page }) => {
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos } }));
  await page.goto('/todos');

  await expect(page.getByText('3 todos on the list.')).toBeVisible();
  await expect(page.getByRole('main').getByRole('listitem')).toHaveText(todos.map((todo) => todo.title));
  await expect(page).toHaveTitle('Todos · Bakhyard');
  await expect(page.getByRole('link', { name: 'Todos', exact: true })).toHaveAttribute('aria-current', 'page');
});

test('a path under /todos shows the Todos page', async ({ page }) => {
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos } }));
  await page.goto('/todos/42');

  await expect(page.getByText('3 todos on the list.')).toBeVisible();
  await expect(page.getByRole('link', { name: 'Todos', exact: true })).toHaveAttribute('aria-current', 'page');
});

test('the Todos page loads the list again each time it shows', async ({ page }) => {
  let answer = todos.slice(1);
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos: answer } }));
  await page.goto('/todos');
  await expect(page.getByText('2 todos on the list.')).toBeVisible();

  await page.getByRole('link', { name: 'Home', exact: true }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Good morning, Alex.' })).toBeVisible();
  // Another device adds a todo while the page is away.
  answer = todos;
  await page.getByRole('link', { name: 'Todos', exact: true }).click();

  await expect(page.getByText('3 todos on the list.')).toBeVisible();
  await expect(page.getByRole('main').getByRole('listitem').first()).toHaveText(todos[0].title);
});

test('a long title wraps on a narrow screen', async ({ page }) => {
  const title = `Read ${'x'.repeat(200)}`;
  await page.setViewportSize({ width: 320, height: 640 });
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos: [{ id: 1, title }] } }));
  await page.goto('/todos');

  await expect(page.getByRole('main').getByRole('listitem')).toHaveText(title);
  const scroll = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(scroll).toBe(0);
});

test('the Todos page says when the todos could not load', async ({ page }) => {
  await page.route('/api/todos', (route) => route.fulfill({ status: 500, body: 'Internal Server Error\n' }));
  await page.goto('/todos');

  await expect(page.getByText("Couldn't load the todos.")).toBeVisible();
  await expect(page.getByText(/on the list\.$/)).toBeHidden();
});
