import type { Page, Route } from '@playwright/test';

import { home, todos } from './demo';
import { expect, test } from './fixtures';

type Answer = Parameters<Route['fulfill']>[0];

/** Answers GET /api/todos with the demo list and each tick with tick's answer; gives the ticks sent, in order. */
async function serveTicks(page: Page, tick: (id: number, done: boolean) => Answer | Promise<Answer>) {
  const sent: string[] = [];
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos } }));
  await page.route('/api/todos/*', async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    sent.push(`${request.method()} ${path} ${request.postData()}`);
    const { done } = request.postDataJSON() as { done: boolean };
    await route.fulfill(await tick(Number(path.split('/').pop()), done));
  });
  return sent;
}

function saved(id: number, done: boolean): Answer {
  return { json: { ...todos.find((todo) => todo.id === id), done } };
}

const [notes, car, passport] = todos;

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test('a click on the box ticks the todo, and another unticks it', async ({ page }) => {
  const sent = await serveTicks(page, saved);
  await page.goto('/todos');
  const box = page.getByRole('checkbox', { name: notes.title });
  const title = page.getByRole('main').getByText(notes.title);
  await expect(page.getByRole('checkbox', { name: passport.title })).toBeChecked();
  await expect(page.getByRole('main').getByText(passport.title)).toHaveCSS('text-decoration-line', 'line-through');
  await expect(title).toHaveCSS('text-decoration-line', 'none');

  await box.click();
  await expect(box).toBeChecked();
  await expect(title).toHaveCSS('text-decoration-line', 'line-through');
  // The box is aria-disabled until the service has saved the tick.
  await expect(box).toHaveAttribute('aria-disabled', 'false');
  await box.click();

  await expect(box).not.toBeChecked();
  await expect(title).toHaveCSS('text-decoration-line', 'none');
  await expect(page.getByRole('main').getByRole('listitem').locator('label')).toHaveText(
    todos.map((todo) => todo.title)
  );
  await expect.poll(() => sent).toEqual(['PATCH /api/todos/3 {"done":true}', 'PATCH /api/todos/3 {"done":false}']);
});

test('a click on the title ticks the todo', async ({ page }) => {
  const sent = await serveTicks(page, saved);
  await page.goto('/todos');

  await page.getByRole('main').getByText(car.title).click();

  await expect(page.getByRole('checkbox', { name: car.title })).toBeChecked();
  await expect.poll(() => sent).toEqual(['PATCH /api/todos/2 {"done":true}']);
});

test('Space on the focused box ticks the todo', async ({ page }) => {
  const sent = await serveTicks(page, saved);
  await page.goto('/todos');
  const box = page.getByRole('checkbox', { name: notes.title });

  await box.press('Space');

  await expect(box).toBeChecked();
  await expect(box).toBeFocused();
  await expect.poll(() => sent).toEqual(['PATCH /api/todos/3 {"done":true}']);
});

test('a failed tick puts the box back and says so, until the next tick', async ({ page }) => {
  let failing = true;
  await serveTicks(page, (id, done) => (failing ? { status: 500, body: 'Internal Server Error\n' } : saved(id, done)));
  await page.goto('/todos');
  const row = page.getByRole('main').getByRole('listitem').filter({ hasText: notes.title });
  const box = row.getByRole('checkbox');

  await box.click();

  await expect(row.getByRole('status')).toHaveText('Tick failed');
  await expect(box).not.toBeChecked();
  await expect(row.getByText(notes.title)).toHaveCSS('text-decoration-line', 'none');
  failing = false;
  await box.click();
  await expect(row.getByRole('status')).toHaveText('');
  await expect(box).toBeChecked();
});

test('a failed untick puts the box back and says so', async ({ page }) => {
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos } }));
  await page.route('/api/todos/*', (route) => route.abort());
  await page.goto('/todos');
  const row = page.getByRole('main').getByRole('listitem').filter({ hasText: passport.title });

  await row.getByRole('checkbox').click();

  await expect(row.getByRole('status')).toHaveText('Untick failed');
  await expect(row.getByRole('checkbox')).toBeChecked();
  await expect(row.getByText(passport.title)).toHaveCSS('text-decoration-line', 'line-through');
});

test('a todo ticks once at a time, and the others meanwhile', async ({ page }) => {
  const answer = Promise.withResolvers<void>();
  const sent = await serveTicks(page, async (id, done) => {
    if (id === notes.id) {
      await answer.promise;
    }
    return saved(id, done);
  });
  await page.goto('/todos');
  const box = page.getByRole('checkbox', { name: notes.title });

  await box.click();
  await expect(box).toHaveAttribute('aria-disabled', 'true');
  // Playwright's click() waits for an aria-disabled box to be enabled, so the click is dispatched, as element.click()
  // does. Space doesn't wait.
  await box.dispatchEvent('click');
  await expect(box).toBeChecked();
  await box.press('Space');
  await expect(box).toBeChecked();
  await page.getByRole('checkbox', { name: car.title }).click();
  answer.resolve();

  await expect(box).toHaveAttribute('aria-disabled', 'false');
  await expect(box).toBeChecked();
  await expect(page.getByRole('checkbox', { name: car.title })).toBeChecked();
  await expect.poll(() => sent).toEqual(['PATCH /api/todos/3 {"done":true}', 'PATCH /api/todos/2 {"done":true}']);
});
