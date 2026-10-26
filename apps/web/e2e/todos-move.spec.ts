import type { Page, Route } from '@playwright/test';

import type { Todo } from '../src/todos/api';
import { home } from './demo';
import { expect, test } from './fixtures';
import { group, heads } from './groups';
import { aTodo, jiraOff } from './jira';

type Answer = Parameters<Route['fulfill']>[0];

/** A made-up user's todos in the service's order: three of normal priority, with a high one among them. */
const list: Todo[] = [
  { id: 5, title: 'Book the car in for its winter tyres', done: false, priority: 'normal' },
  { id: 4, title: 'Write the release notes for 2.4', done: false, priority: 'high' },
  { id: 3, title: 'Pick up the dry cleaning', done: false, priority: 'normal' },
  { id: 2, title: 'Call the plumber about the boiler', done: false, priority: 'normal' },
  { id: 1, title: 'Renew the passport', done: true, priority: 'low' },
];
const [car, notes, cleaning, plumber, passport] = list;

/** Answers GET /api/todos with the list and each PATCH with the answer, else the todo; gives the PATCHes sent. */
async function serveMoves(page: Page, answer?: () => Answer | Promise<Answer>) {
  const sent: string[] = [];
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos: list } }));
  await page.route(aTodo, async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    sent.push(`${request.method()} ${path} ${request.postData()}`);
    const todo = list.find(({ id }) => id === Number(path.split('/').pop()));
    await route.fulfill(answer ? await answer() : { json: todo });
  });
  return sent;
}

/** The Normal group's rows in the page's order: each todo's title, or what the open editor's field holds. */
function normal(page: Page) {
  return group(page, 'Normal priority')
    .getByRole('listitem')
    .evaluateAll((rows) =>
      rows.map(
        (row) =>
          row.querySelector<HTMLInputElement>('input[type="text"]')?.value ??
          row.querySelector(':scope > label')?.textContent?.trim()
      )
    );
}

/** The open editor's status line; each row has a status line of its own too. */
function editorStatus(page: Page) {
  return page.getByRole('main').getByRole('list').locator('form').getByRole('status');
}

const button = (page: Page, name: string) => page.getByRole('button', { name, exact: true });

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await jiraOff(page);
});

test('Move down and Move up move the todo a step in its group, past the todos of others', async ({ page }) => {
  const sent = await serveMoves(page);
  await page.goto('/todos');

  await page.getByRole('button', { name: `Edit ${car.title}` }).click();
  await button(page, 'Move down').click();
  await expect(editorStatus(page)).toHaveText('Moved down');
  await expect.poll(() => normal(page)).toEqual([cleaning.title, car.title, plumber.title]);
  await expect(button(page, 'Move down')).toBeFocused();
  await button(page, 'Move down').click();
  await expect.poll(() => normal(page)).toEqual([cleaning.title, plumber.title, car.title]);
  await expect(button(page, 'Move down')).toHaveAttribute('aria-disabled', 'true');
  await button(page, 'Move up').click();

  await expect(editorStatus(page)).toHaveText('Moved up');
  await expect.poll(() => normal(page)).toEqual([cleaning.title, car.title, plumber.title]);
  await expect(heads(page)).toHaveText(['High priority', 'Normal priority', 'Low priority']);
  await expect(group(page, 'High priority').getByRole('listitem').locator('label')).toHaveText([notes.title]);
  expect(sent).toEqual([
    'PATCH /api/todos/5 {"after":3}',
    'PATCH /api/todos/5 {"after":2}',
    'PATCH /api/todos/5 {"before":2}',
  ]);
});

test('the focus stays on the key pressed, and the title and priority picked wait for Save', async ({ page }) => {
  const sent = await serveMoves(page);
  await page.goto('/todos');
  const moveDown = button(page, 'Move down');

  await page.getByRole('button', { name: `Edit ${car.title}` }).click();
  await page.getByRole('textbox', { name: 'Title' }).fill('Book the car in for its MOT');
  await page.getByRole('radio', { name: 'High' }).click();
  await moveDown.focus();
  await page.keyboard.press('Enter');
  await expect.poll(() => normal(page)).toEqual([cleaning.title, 'Book the car in for its MOT', plumber.title]);
  await expect(moveDown).toBeFocused();
  await page.keyboard.press('Enter');
  await expect.poll(() => normal(page)).toEqual([cleaning.title, plumber.title, 'Book the car in for its MOT']);
  await expect(moveDown).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('radio', { name: 'High' })).toBeChecked();
  await button(page, 'Cancel').click();

  await expect.poll(() => normal(page)).toEqual([cleaning.title, plumber.title, car.title]);
  expect(sent).toEqual(['PATCH /api/todos/5 {"after":3}', 'PATCH /api/todos/5 {"after":2}']);
});

test("the moves past a group's ends are muted and send nothing", async ({ page }) => {
  const sent = await serveMoves(page);
  await page.goto('/todos');

  await page.getByRole('button', { name: `Edit ${car.title}` }).click();
  await expect(button(page, 'Move up')).toHaveAttribute('aria-disabled', 'true');
  await expect(button(page, 'Move down')).toHaveAttribute('aria-disabled', 'false');
  const muted = await button(page, 'Move up').evaluate((element) => getComputedStyle(element).color);
  await expect(button(page, 'Move down')).not.toHaveCSS('color', muted);
  await button(page, 'Move up').dispatchEvent('click');
  await button(page, 'Cancel').click();
  await page.getByRole('button', { name: `Edit ${passport.title}` }).click();
  for (const name of ['Move up', 'Move down']) {
    await expect(button(page, name)).toHaveAttribute('aria-disabled', 'true');
    await expect(button(page, name)).toHaveCSS('color', muted);
    await button(page, name).dispatchEvent('click');
  }

  await expect(editorStatus(page)).toHaveText('');
  expect(sent).toEqual([]);
});

test('while a move runs, the editor says so, and it and the other rows wait', async ({ page }) => {
  const answer = Promise.withResolvers<Answer>();
  const sent = await serveMoves(page, () => answer.promise);
  await page.goto('/todos');
  const field = page.getByRole('textbox', { name: 'Title' });

  await page.getByRole('button', { name: `Edit ${car.title}` }).click();
  await field.fill('Book the car in for its MOT');
  await button(page, 'Move down').click();
  await expect(editorStatus(page)).toHaveText('Moving…');
  const waiting = ['Save', 'Cancel', 'Delete', 'Move up', 'Move down', `Edit ${notes.title}`];
  for (const name of waiting) {
    await expect(button(page, name)).toHaveAttribute('aria-disabled', 'true');
  }
  for (const name of ['High', 'Normal', 'Low']) {
    await expect(page.getByRole('radio', { name })).toHaveAttribute('aria-disabled', 'true');
  }
  await expect(field).toHaveAttribute('readonly');
  for (const name of waiting) {
    await button(page, name).dispatchEvent('click');
  }
  await field.press('Escape');
  answer.resolve({ json: car });

  await expect(editorStatus(page)).toHaveText('Moved down');
  await expect(page.getByRole('alertdialog')).toBeHidden();
  await expect.poll(() => normal(page)).toEqual([cleaning.title, 'Book the car in for its MOT', plumber.title]);
  await expect(field).not.toHaveAttribute('readonly');
  expect(sent).toEqual(['PATCH /api/todos/5 {"after":3}']);
});

test('a failed move says so and leaves the list as it was', async ({ page }) => {
  const sent = await serveMoves(page, () => ({ status: 500 }));
  await page.goto('/todos');

  await page.getByRole('button', { name: `Edit ${plumber.title}` }).click();
  await button(page, 'Move up').click();

  await expect(editorStatus(page)).toHaveText('Move failed');
  await expect.poll(() => normal(page)).toEqual([car.title, cleaning.title, plumber.title]);
  await expect(button(page, 'Move up')).toHaveAttribute('aria-disabled', 'false');
  expect(sent).toEqual(['PATCH /api/todos/2 {"before":3}']);
});

test('the moves wrap under the other buttons first, and share their line once it has room', async ({ page }) => {
  await serveMoves(page);
  await page.goto('/todos');
  await page.getByRole('button', { name: `Edit ${car.title}` }).click();
  const controls = [
    page.getByRole('textbox', { name: 'Title' }),
    ...['Delete', 'Move up', 'Move down'].map((name) => button(page, name)),
  ];
  const bottom = (box: { y: number; height: number } | null) => (box?.y ?? 0) + (box?.height ?? 0);

  for (const [width, place] of [
    [320, 'under Delete'],
    [440, 'under Delete'],
    [1024, 'under the field'],
    [1280, 'beside the field'],
  ] as const) {
    await page.setViewportSize({ width, height: 800 });
    const [field, remove, up, down] = await Promise.all(controls.map((control) => control.boundingBox()));
    const want = {
      'under Delete': bottom(remove) + 12,
      'under the field': bottom(field) + 12,
      'beside the field': field?.y,
    };
    expect(up?.y, `Move up at ${width} px, ${place}`).toBe(want[place]);
    expect(down?.y, `Move down at ${width} px`).toBe(up?.y);
    const scroll = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth
    );
    expect(scroll, `sideways scroll at ${width} px`).toBe(0);
  }
});
