import type { Locator, Page, Route } from '@playwright/test';

import type { Todo } from '../src/todos/api';
import { home } from './demo';
import { expect, test } from './fixtures';
import { group } from './groups';

type Answer = Parameters<Route['fulfill']>[0];

/** A made-up user's todos in the service's order: three of normal priority, with a high one among them. */
const list: Todo[] = [
  { id: 5, title: 'Book the car in for its winter tyres', done: false, priority: 'normal' },
  { id: 4, title: 'Write the release notes for 2.4', done: false, priority: 'high' },
  { id: 3, title: 'Pick up the dry cleaning', done: false, priority: 'normal' },
  { id: 2, title: 'Call the plumber about the boiler', done: false, priority: 'normal' },
  { id: 1, title: 'Renew the passport', done: true, priority: 'low' },
];
const [car, notes, cleaning, plumber] = list;

/** Answers GET /api/todos with the list and each PATCH with the answer, else the todo; gives the PATCHes sent. */
async function serveDrops(page: Page, answer?: () => Answer | Promise<Answer>) {
  const sent: string[] = [];
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos: list } }));
  await page.route('/api/todos/*', async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    sent.push(`${request.method()} ${path} ${request.postData()}`);
    const todo = list.find(({ id }) => id === Number(path.split('/').pop()));
    await route.fulfill(answer ? await answer() : { json: todo });
  });
  return sent;
}

/** The titles in the group with the head, in the page's order. */
function titles(page: Page, head: string) {
  return group(page, head).getByRole('listitem').locator(':scope > label');
}

const row = (page: Page, todo: Todo) => page.getByRole('listitem').filter({ hasText: todo.title });
const grip = (page: Page, todo: Todo) => row(page, todo).locator('.grip');

/** Where in the viewport the box's point a share of the way down it is, rounded to a whole px. */
async function point(locator: Locator, share: number) {
  const box = await locator.boundingBox();
  if (box === null) {
    throw new Error(`no box for ${locator}`);
  }
  return { x: Math.round(box.x + box.width / 2), y: Math.round(box.y + box.height * share) };
}

/**
 * Presses the mouse on the todo's grip, with its row in the middle of the screen, clear of the phone's tab bar, and
 * moves it in steps across the titles, as a hand would, to the middle of the target's row a share of the way down,
 * without letting go. Gives the point pressed.
 */
async function lift(page: Page, todo: Todo, target: Todo, share: number) {
  await row(page, todo).evaluate((element) => element.scrollIntoView({ block: 'center' }));
  const from = await point(grip(page, todo), 0.25);
  const to = await point(row(page, target), share);
  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  await page.mouse.move(to.x, to.y, { steps: 8 });
  return from;
}

/** Drags the todo by its grip to the point a share of the way down the target's row, once the grip no longer waits. */
async function drag(page: Page, todo: Todo, target: Todo, share: number) {
  await expect(grip(page, todo)).toHaveCSS('cursor', 'grab');
  await lift(page, todo, target, share);
  await page.mouse.up();
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test('dragging a todo by its grip moves it in its group, past the todos of others', async ({ page }) => {
  const sent = await serveDrops(page);
  await page.goto('/todos');

  await drag(page, car, plumber, 0.75);
  await expect(titles(page, 'Normal priority')).toHaveText([cleaning.title, plumber.title, car.title]);
  await drag(page, plumber, cleaning, 0.25);

  await expect(titles(page, 'Normal priority')).toHaveText([plumber.title, cleaning.title, car.title]);
  await expect(titles(page, 'High priority')).toHaveText([notes.title]);
  expect(sent).toEqual(['PATCH /api/todos/5 {"after":2}', 'PATCH /api/todos/2 {"before":3}']);
  expect(await page.evaluate(() => getSelection()?.toString())).toBe('');
});

test('dropping a todo in another group gives it that priority, and the next drag finds it there', async ({ page }) => {
  const sent = await serveDrops(page);
  await page.goto('/todos');

  await drag(page, cleaning, notes, 0.75);

  await expect(titles(page, 'High priority')).toHaveText([notes.title, cleaning.title]);
  await expect(titles(page, 'Normal priority')).toHaveText([car.title, plumber.title]);
  await expect(group(page, 'High priority')).toContainText('2 todos');
  await drag(page, car, cleaning, 0.75);
  await expect(titles(page, 'High priority')).toHaveText([notes.title, cleaning.title, car.title]);
  expect(sent).toEqual([
    'PATCH /api/todos/3 {"after":4,"priority":"high"}',
    'PATCH /api/todos/5 {"after":3,"priority":"high"}',
  ]);
});

test('the dragged row stays in place, lifted, and a line follows the pointer to where the todo would land', async ({
  page,
}) => {
  await serveDrops(page);
  await page.goto('/todos');
  await row(page, car).evaluate((element) => element.scrollIntoView({ block: 'center' }));
  const rest = await row(page, car).boundingBox();

  const from = await lift(page, car, plumber, 0.75);

  expect(await row(page, car).boundingBox(), 'in place').toEqual(rest);
  await expect(row(page, car)).toHaveCSS('outline-style', 'solid');
  await expect(row(page, car)).toHaveCSS('box-shadow', / 8px 8px 0px 0px$/);
  await expect(grip(page, car)).toHaveCSS('cursor', 'grabbing');
  await expect(row(page, plumber)).toHaveCSS('box-shadow', / 0px -2px 0px 0px inset$/);
  const covered = await row(page, plumber).evaluate((element) => {
    const box = element.getBoundingClientRect();
    return !element.contains(document.elementFromPoint(box.x + box.width / 2, box.bottom - 1));
  });
  expect(covered, 'the line in sight').toBe(false);
  await expect(row(page, cleaning)).toHaveCSS('box-shadow', 'none');
  await page.mouse.move(from.x, (await point(row(page, plumber), 0.25)).y, { steps: 4 });
  await expect(row(page, plumber)).toHaveCSS('box-shadow', / 0px 2px 0px 0px inset$/);
  await page.mouse.move(from.x, from.y, { steps: 4 });
  await expect(row(page, plumber), 'no line over its own place').toHaveCSS('box-shadow', 'none');
  await expect(row(page, car), 'not a hovered row').toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
  await page.mouse.up();
  await expect(row(page, car)).toHaveCSS('outline-style', 'none');
});

test('a todo dropped back next to its own place stays, and nothing is sent', async ({ page }) => {
  const sent = await serveDrops(page);
  await page.goto('/todos');

  await drag(page, cleaning, cleaning, 0.75);
  await drag(page, cleaning, car, 0.75);

  await expect(titles(page, 'Normal priority')).toHaveText([car.title, cleaning.title, plumber.title]);
  expect(sent).toEqual([]);
});

test('a dropped todo moves before the answer, and the grips wait for it', async ({ page }) => {
  const answer = Promise.withResolvers<Answer>();
  const sent = await serveDrops(page, () => answer.promise);
  await page.goto('/todos');

  await drag(page, car, plumber, 0.75);
  await expect(titles(page, 'Normal priority')).toHaveText([cleaning.title, plumber.title, car.title]);
  await expect(grip(page, plumber)).toHaveCSS('cursor', 'progress');
  await lift(page, plumber, cleaning, 0.25);
  await expect(row(page, plumber), 'not lifted').toHaveCSS('outline-style', 'none');
  await page.mouse.up();
  answer.resolve({ json: car });

  await drag(page, plumber, cleaning, 0.25);
  await expect(titles(page, 'Normal priority')).toHaveText([plumber.title, cleaning.title, car.title]);
  expect(sent).toEqual(['PATCH /api/todos/5 {"after":2}', 'PATCH /api/todos/2 {"before":3}']);
});

test('the grips wait while the editor saves', async ({ page }) => {
  const answer = Promise.withResolvers<Answer>();
  const sent = await serveDrops(page, () => answer.promise);
  await page.goto('/todos');

  await page.getByRole('button', { name: `Edit ${car.title}` }).click();
  await page.getByRole('textbox', { name: 'Title' }).fill('Book the car in');
  await page.getByRole('button', { name: 'Save' }).click();
  await expect(grip(page, plumber)).toHaveCSS('cursor', 'progress');
  await lift(page, plumber, cleaning, 0.25);
  await expect(row(page, plumber), 'not lifted').toHaveCSS('outline-style', 'none');
  await page.mouse.up();
  answer.resolve({ json: { ...car, title: 'Book the car in' } });

  await expect(grip(page, plumber)).toHaveCSS('cursor', 'grab');
  expect(sent).toEqual(['PATCH /api/todos/5 {"title":"Book the car in"}']);
});

test('a failed drop puts the todo back and says so on its row', async ({ page }) => {
  const sent = await serveDrops(page, () => ({ status: 500 }));
  await page.goto('/todos');

  await drag(page, car, notes, 0.75);

  await expect(row(page, car).getByRole('status')).toHaveText('Move failed');
  await expect(titles(page, 'Normal priority')).toHaveText([car.title, cleaning.title, plumber.title]);
  await expect(titles(page, 'High priority')).toHaveText([notes.title]);
  expect(sent).toEqual(['PATCH /api/todos/5 {"after":4,"priority":"high"}']);
});

/**
 * Puts a finger on the todo's grip and moves it in steps to the point a share of the way down the target's row, without
 * lifting it. Gives the DevTools session that sends the touches, which only Chromium takes.
 */
async function touch(page: Page, todo: Todo, target: Todo, share: number) {
  const from = await point(grip(page, todo), 0.25);
  const to = await point(row(page, target), share);
  const finger = await page.context().newCDPSession(page);
  await finger.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [from] });
  for (let step = 1; step <= 8; step++) {
    const y = from.y + ((to.y - from.y) * step) / 8;
    await finger.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: from.x, y }] });
  }
  return finger;
}

test('a finger drags a todo by its grip without scrolling the page', async ({ page, browserName }) => {
  // Only Chromium takes touches from a test, through its DevTools protocol; WebKit's are taps only.
  test.skip(browserName !== 'chromium');
  const sent = await serveDrops(page);
  await page.goto('/todos');
  await row(page, car).evaluate((element) => element.scrollIntoView({ block: 'center' }));
  const scrolled = await page.evaluate(() => scrollY);

  const finger = await touch(page, car, plumber, 0.75);
  await finger.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });

  await expect(titles(page, 'Normal priority')).toHaveText([cleaning.title, plumber.title, car.title]);
  expect(sent).toEqual(['PATCH /api/todos/5 {"after":2}']);
  expect(await page.evaluate(() => scrollY)).toBe(scrolled);
});

test('a touch the browser cancels lays the row down and drops nothing', async ({ page, browserName }) => {
  // As above: only Chromium takes touches from a test.
  test.skip(browserName !== 'chromium');
  const sent = await serveDrops(page);
  await page.goto('/todos');
  await row(page, car).evaluate((element) => element.scrollIntoView({ block: 'center' }));

  const finger = await touch(page, car, plumber, 0.75);
  await expect(row(page, plumber)).toHaveCSS('box-shadow', / 0px -2px 0px 0px inset$/);
  await finger.send('Input.dispatchTouchEvent', { type: 'touchCancel', touchPoints: [] });

  await expect(row(page, car)).toHaveCSS('outline-style', 'none');
  await expect(row(page, plumber)).toHaveCSS('box-shadow', 'none');
  await expect(titles(page, 'Normal priority')).toHaveText([car.title, cleaning.title, plumber.title]);
  expect(sent).toEqual([]);
});
