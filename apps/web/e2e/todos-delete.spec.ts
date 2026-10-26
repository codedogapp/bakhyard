import type { Page, Route } from '@playwright/test';

import type { Todo } from '../src/todos/api';
import { home, todos } from './demo';
import { expect, test } from './fixtures';
import { aTodo, jiraOff } from './jira';

type Answer = Parameters<Route['fulfill']>[0];

/** Answers GET /api/todos with the list and each DELETE with delete's answer; gives the DELETEs sent, in order. */
async function serveDeletes(page: Page, answer: () => Answer | Promise<Answer>, list: Todo[] = todos) {
  const sent: string[] = [];
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos: list } }));
  await page.route(aTodo, async (route) => {
    sent.push(`${route.request().method()} ${new URL(route.request().url()).pathname}`);
    await route.fulfill(await answer());
  });
  return sent;
}

const deleted: Answer = { status: 204 };

/** The open editor's status line; each row has a status line of its own too. */
function editorStatus(page: Page) {
  return page.getByRole('main').getByRole('list').locator('form').getByRole('status');
}

/** The open editor's Delete button, which asks in the dialog. */
function editorDelete(page: Page) {
  return page.getByRole('main').getByRole('listitem').getByRole('button', { name: 'Delete' });
}

const [notes, car, passport] = todos;
const dialogName = 'Delete this todo?';

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await jiraOff(page);
});

test('Delete asks in a modal dialog that names the todo as stored, with the focus on Cancel', async ({ page }) => {
  const sent = await serveDeletes(page, () => deleted);
  await page.goto('/todos');

  await page.getByRole('button', { name: `Edit ${car.title}` }).click();
  await page.getByRole('textbox', { name: 'Title' }).fill('Book the car in for its MOT');
  await editorDelete(page).click();

  const dialog = page.getByRole('alertdialog', { name: dialogName });
  await expect(dialog).toHaveAccessibleDescription(`“${car.title}” will be deleted. This can't be undone.`);
  expect(await dialog.evaluate((element) => element.matches(':modal'))).toBe(true);
  await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeFocused();
  expect(sent).toEqual([]);
});

test('Cancel and Escape close the dialog with no request, and give the focus back to Delete', async ({ page }) => {
  const sent = await serveDeletes(page, () => deleted);
  await page.goto('/todos');
  const field = page.getByRole('textbox', { name: 'Title' });
  const dialog = page.getByRole('alertdialog', { name: dialogName });

  await page.getByRole('button', { name: `Edit ${car.title}` }).click();
  await field.fill('Book the car in for its MOT');
  // A click, which in WebKit focuses neither Delete nor Cancel, so the browser has no focus to give back.
  await editorDelete(page).click();
  await dialog.getByRole('button', { name: 'Cancel' }).click();
  await expect(dialog).toBeHidden();
  await expect(editorDelete(page)).toBeFocused();
  await editorDelete(page).press('Enter');
  await expect(dialog).toBeVisible();
  await page.keyboard.press('Escape');

  await expect(dialog).toBeHidden();
  await expect(editorDelete(page)).toBeFocused();
  await expect(field).toHaveValue('Book the car in for its MOT');
  expect(sent).toEqual([]);
});

test('Delete takes the todo off the list, and the focus goes to the next row, else the previous', async ({ page }) => {
  const sent = await serveDeletes(page, () => deleted);
  await page.goto('/todos');

  for (const [todo, next] of [
    [car, passport],
    [passport, notes],
  ]) {
    await page.getByRole('button', { name: `Edit ${todo.title}` }).click();
    await editorDelete(page).click();
    await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click();
    await expect(page.getByRole('button', { name: `Edit ${next.title}` })).toBeFocused();
  }

  await expect(page.getByRole('main').getByRole('listitem').locator('label')).toHaveText([notes.title]);
  await expect(page.getByText('1 todo on the list.')).toBeVisible();
  expect(sent).toEqual(['DELETE /api/todos/2', 'DELETE /api/todos/1']);
});

test('a 404 counts as deleted, and the Add field takes the focus once no todo is left', async ({ page }) => {
  const sent = await serveDeletes(page, () => ({ status: 404, body: 'no such todo\n' }), [notes]);
  await page.goto('/todos');

  await page.getByRole('button', { name: `Edit ${notes.title}` }).click();
  await editorDelete(page).click();
  await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click();

  await expect(page.getByRole('textbox', { name: 'New todo' })).toBeFocused();
  await expect(page.getByText('No todos on the list.')).toBeVisible();
  await expect(page.getByRole('main').getByRole('list')).toBeHidden();
  expect(sent).toEqual(['DELETE /api/todos/3']);
});

test('while a delete runs, the editor says so, and it and the other rows wait', async ({ page }) => {
  const answer = Promise.withResolvers<Answer>();
  const sent = await serveDeletes(page, () => answer.promise);
  await page.goto('/todos');
  const field = page.getByRole('textbox', { name: 'Title' });
  const dialog = page.getByRole('alertdialog', { name: dialogName });

  await page.getByRole('button', { name: `Edit ${car.title}` }).click();
  await editorDelete(page).click();
  await dialog.getByRole('button', { name: 'Delete' }).click();
  await expect(editorStatus(page)).toHaveText('Deleting…');
  await expect(editorDelete(page)).toBeFocused();
  await expect(field).not.toBeEditable();
  for (const name of ['Save', 'Cancel', 'Delete', `Edit ${notes.title}`]) {
    const button = page.getByRole('main').getByRole('button', { name, exact: true });
    await expect(button).toHaveAttribute('aria-disabled', 'true');
    await button.dispatchEvent('click');
  }
  await field.press('Escape');
  await expect(dialog).toBeHidden();
  await expect(field).toBeVisible();
  answer.resolve(deleted);

  await expect(page.getByRole('main').getByRole('listitem').locator('label')).toHaveText([notes.title, passport.title]);
  await expect(page.getByRole('button', { name: `Edit ${notes.title}` })).toHaveAttribute('aria-disabled', 'false');
  expect(sent).toEqual(['DELETE /api/todos/2']);
});

test('a failed delete says so and keeps the editor, and the next try deletes', async ({ page }) => {
  let failing = true;
  const sent = await serveDeletes(page, () => (failing ? { status: 502, body: 'Bad Gateway\n' } : deleted));
  await page.goto('/todos');
  const field = page.getByRole('textbox', { name: 'Title' });
  const confirm = page.getByRole('alertdialog').getByRole('button', { name: 'Delete' });

  await page.getByRole('button', { name: `Edit ${car.title}` }).click();
  await field.fill('Book the car in for its MOT');
  await editorDelete(page).click();
  await confirm.click();
  await expect(editorStatus(page)).toHaveText('Delete failed');
  await expect(field).toHaveValue('Book the car in for its MOT');
  await expect(field).toBeEditable();
  failing = false;
  await editorDelete(page).click();
  await confirm.click();

  await expect(page.getByRole('main').getByRole('listitem').locator('label')).toHaveText([notes.title, passport.title]);
  expect(sent).toEqual(['DELETE /api/todos/2', 'DELETE /api/todos/2']);
});

test('Delete wraps under Save on the narrowest phones, and the dialog fits a 320 px screen, long title and all', async ({
  page,
}) => {
  const long = { ...car, title: `Renew the ${'passport'.repeat(8)}` };
  await serveDeletes(page, () => deleted, [notes, long, passport]);
  await page.goto('/todos');
  await page.getByRole('button', { name: `Edit ${long.title}` }).click();
  const save = page.getByRole('button', { name: 'Save' });

  for (const [width, wraps] of [
    [320, true],
    [440, false],
  ] as const) {
    await page.setViewportSize({ width, height: 800 });
    const [saveBox, deleteBox] = await Promise.all([save.boundingBox(), editorDelete(page).boundingBox()]);
    const under = (saveBox?.y ?? 0) + (saveBox?.height ?? 0) + 12;
    expect(deleteBox?.y, `Delete at ${width} px`).toBe(wraps ? under : saveBox?.y);
  }
  await page.setViewportSize({ width: 320, height: 800 });
  await editorDelete(page).click();
  const dialog = page.getByRole('alertdialog', { name: dialogName });

  const [box, cancel, confirm] = await Promise.all(
    [dialog, dialog.getByRole('button', { name: 'Cancel' }), dialog.getByRole('button', { name: 'Delete' })].map(
      (element) => element.boundingBox()
    )
  );
  expect(box).toMatchObject({ x: 20, width: 280 });
  expect(confirm?.y).toBe(cancel?.y);
  expect(confirm?.x).toBeCloseTo((cancel?.x ?? 0) + (cancel?.width ?? 0) + 12, 1);
  expect(await dialog.evaluate((element) => element.scrollWidth - element.clientWidth)).toBe(0);
  const scroll = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(scroll).toBe(0);
});

/** A `color(srgb r g b / a)` value as 0 to 255 channels and the alpha: each browser prints its own precision. */
function srgb(value: string): number[] {
  const [red, green, blue, alpha] = value.match(/[\d.]+/g)?.map(Number) ?? [];
  return [...[red, green, blue].map((channel) => Math.round(channel * 255)), alpha];
}

const rgb = (channels: readonly number[]) => `rgb(${channels.join(', ')})`;

test('the dialog veils the page in its paper colour, and its Delete is ink, in either theme', async ({ page }) => {
  await serveDeletes(page, () => deleted);
  const themes = [
    { scheme: 'light', paper: [250, 250, 248], ink: [17, 17, 17] },
    { scheme: 'dark', paper: [19, 19, 18], ink: [237, 237, 234] },
  ] as const;

  for (const { scheme, paper, ink } of themes) {
    await page.emulateMedia({ colorScheme: scheme });
    await page.goto('/todos');
    await page.getByRole('button', { name: `Edit ${car.title}` }).click();
    await editorDelete(page).click();
    const dialog = page.getByRole('alertdialog', { name: dialogName });
    const confirm = dialog.getByRole('button', { name: 'Delete' });

    const backdrop = await dialog.evaluate((element) => getComputedStyle(element, '::backdrop').backgroundColor);
    expect(srgb(backdrop), scheme).toEqual([...paper, 0.75]);
    await expect(confirm).toHaveCSS('background-color', rgb(ink));
    await expect(confirm).toHaveCSS('color', rgb(paper));
    await confirm.hover();
    await expect(confirm).toHaveCSS('background-color', rgb(paper));
    await expect(confirm).toHaveCSS('color', rgb(ink));
  }
});
