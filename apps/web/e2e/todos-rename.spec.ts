import type { Page, Route } from '@playwright/test';

import { home, todos } from './demo';
import { expect, test } from './fixtures';
import { aTodo, jiraOff } from './jira';

type Answer = Parameters<Route['fulfill']>[0];

/** Answers GET /api/todos with the demo list and each PATCH with rename's answer; gives the PATCHes sent, in order. */
async function serveRenames(page: Page, rename: (id: number, title: string) => Answer | Promise<Answer>) {
  const sent: string[] = [];
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos } }));
  await page.route(aTodo, async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    sent.push(`${request.method()} ${path} ${request.postData()}`);
    const { title } = request.postDataJSON() as { title: string };
    await route.fulfill(await rename(Number(path.split('/').pop()), title));
  });
  return sent;
}

function renamed(id: number, title: string): Answer {
  return { json: { ...todos.find((todo) => todo.id === id), title: title.trim() } };
}

/** The open editor's status line; each row has a status line of its own too. */
function editorStatus(page: Page) {
  return page.getByRole('main').getByRole('list').locator('form').getByRole('status');
}

const [notes, car] = todos;
const titles = todos.map((todo) => todo.title);

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await jiraOff(page);
});

test('Edit opens an editor with the title, and Enter saves the new one', async ({ page }) => {
  const sent = await serveRenames(page, renamed);
  await page.goto('/todos');
  const row = page.getByRole('main').getByRole('listitem').first();

  await row.getByRole('button', { name: `Edit ${notes.title}` }).click();
  const field = row.getByRole('textbox', { name: 'Title' });
  await expect(field).toBeFocused();
  await expect(field).toHaveValue(notes.title);
  await expect(row.getByRole('checkbox')).toHaveCount(0);
  const caret = await field.evaluate((input: HTMLInputElement) => [input.selectionStart, input.selectionEnd]);
  expect(caret).toEqual([notes.title.length, notes.title.length]);
  await field.pressSequentially(' and the blog post');
  await field.press('Enter');

  await expect(row.getByRole('checkbox')).toHaveAccessibleName(`${notes.title} and the blog post`);
  await expect(row.getByRole('button', { name: `Edit ${notes.title} and the blog post` })).toBeFocused();
  await expect(field).toHaveCount(0);
  expect(sent).toEqual([`PATCH /api/todos/3 {"title":"${notes.title} and the blog post"}`]);
});

test('Cancel and Escape close the editor with no request', async ({ page }) => {
  const sent = await serveRenames(page, renamed);
  await page.goto('/todos');
  const edit = page.getByRole('button', { name: `Edit ${car.title}` });
  const field = page.getByRole('textbox', { name: 'Title' });

  await edit.click();
  await field.fill('Book the car in');
  await page.getByRole('button', { name: 'Cancel' }).click();
  await expect(edit).toBeFocused();
  await edit.click();
  await expect(field).toHaveValue(car.title);
  await field.press('Escape');

  await expect(edit).toBeFocused();
  await expect(page.getByRole('main').getByRole('listitem').locator('label')).toHaveText(titles);
  expect(sent).toEqual([]);
});

test('one editor is open at a time', async ({ page }) => {
  const sent = await serveRenames(page, renamed);
  await page.goto('/todos');
  const field = page.getByRole('textbox', { name: 'Title' });

  await page.getByRole('button', { name: `Edit ${notes.title}` }).click();
  await field.fill('Write the blog post');
  await page.getByRole('button', { name: `Edit ${car.title}` }).click();

  await expect(field).toHaveValue(car.title);
  await expect(field).toBeFocused();
  await expect(page.getByRole('checkbox', { name: notes.title })).toBeVisible();
  expect(sent).toEqual([]);
});

test('a title left as it was closes the editor with no request, and a blank one sends nothing', async ({ page }) => {
  const sent = await serveRenames(page, renamed);
  await page.goto('/todos');
  const edit = page.getByRole('button', { name: `Edit ${car.title}` });
  const field = page.getByRole('textbox', { name: 'Title' });

  await edit.click();
  await field.press('Enter');
  await expect(edit).toBeFocused();
  await edit.click();
  await field.fill(' \t ');
  await page.getByRole('button', { name: 'Save' }).click();

  await expect(field).toHaveValue(' \t ');
  await expect(editorStatus(page)).toHaveText('');
  expect(sent).toEqual([]);
});

test('a refused title stays in the editor, with the reason', async ({ page }) => {
  const title = `Read ${'x'.repeat(500)}`;
  await serveRenames(page, () => ({ status: 400, body: 'invalid title: over 500 characters\n' }));
  await page.goto('/todos');
  const field = page.getByRole('textbox', { name: 'Title' });

  await page.getByRole('button', { name: `Edit ${notes.title}` }).click();
  await field.fill(title);
  await field.press('Enter');

  await expect(editorStatus(page)).toHaveText('invalid title: over 500 characters');
  await expect(field).toHaveValue(title);
  await expect(field).toBeEditable();
});

test('a failed save says so, and the next try saves', async ({ page }) => {
  let failing = true;
  const sent = await serveRenames(page, (id, title) =>
    failing ? { status: 502, body: 'Bad Gateway\n' } : renamed(id, title)
  );
  await page.goto('/todos');
  const field = page.getByRole('textbox', { name: 'Title' });

  await page.getByRole('button', { name: `Edit ${car.title}` }).click();
  await field.fill('Book the car in for its MOT');
  await page.getByRole('button', { name: 'Save' }).click();
  await expect(editorStatus(page)).toHaveText('Save failed');
  await expect(field).toHaveValue('Book the car in for its MOT');
  failing = false;
  await page.getByRole('button', { name: 'Save' }).click();

  await expect(page.getByRole('main').getByRole('listitem').nth(1).getByRole('checkbox')).toHaveAccessibleName(
    'Book the car in for its MOT'
  );
  expect(sent).toHaveLength(2);
});

test('while a save runs, the editor waits for it and no other opens', async ({ page }) => {
  const answer = Promise.withResolvers<Answer>();
  const sent = await serveRenames(page, () => answer.promise);
  await page.goto('/todos');
  const field = page.getByRole('textbox', { name: 'Title' });
  const otherEdit = page.getByRole('button', { name: `Edit ${car.title}` });

  await page.getByRole('button', { name: `Edit ${notes.title}` }).click();
  await field.fill('Write the release notes for 2.5');
  await field.press('Enter');
  await expect(editorStatus(page)).toHaveText('Saving…');
  await expect(field).not.toBeEditable();
  for (const button of [page.getByRole('button', { name: 'Save' }), page.getByRole('button', { name: 'Cancel' })]) {
    await expect(button).toHaveAttribute('aria-disabled', 'true');
    await button.dispatchEvent('click');
  }
  await expect(otherEdit).toHaveAttribute('aria-disabled', 'true');
  await otherEdit.dispatchEvent('click');
  await field.press('Escape');
  await field.press('Enter');
  await expect(field).toHaveValue('Write the release notes for 2.5');
  answer.resolve(renamed(notes.id, 'Write the release notes for 2.5'));

  await expect(page.getByRole('main').getByRole('listitem').first().getByRole('checkbox')).toHaveAccessibleName(
    'Write the release notes for 2.5'
  );
  await expect(otherEdit).toHaveAttribute('aria-disabled', 'false');
  expect(sent).toHaveLength(1);
});

test("the row's tick failure stays in sight while its editor is open", async ({ page }) => {
  const answer = Promise.withResolvers<Answer>();
  const sent = await serveRenames(page, () => answer.promise);
  await page.goto('/todos');
  const row = page.getByRole('main').getByRole('listitem').first();

  await row.getByRole('checkbox').click();
  await row.getByRole('button', { name: `Edit ${notes.title}` }).click();
  answer.resolve({ status: 500, body: 'Internal Server Error\n' });

  await expect(row.getByRole('status').filter({ hasText: 'Tick failed' })).toBeVisible();
  await expect(row.getByRole('textbox', { name: 'Title' })).toBeFocused();
  await row.getByRole('textbox', { name: 'Title' }).press('Escape');
  await expect(row.getByRole('checkbox')).not.toBeChecked();
  expect(sent).toEqual(['PATCH /api/todos/3 {"done":true}']);
});

test('the priority and the buttons each wrap under the field on a phone, and sit beside it on a desktop', async ({
  page,
}) => {
  await serveRenames(page, renamed);
  await page.goto('/todos');
  await page.getByRole('button', { name: `Edit ${notes.title}` }).click();
  const controls = [
    page.getByRole('textbox', { name: 'Title' }),
    page.getByRole('group', { name: 'Priority' }),
    page.getByRole('button', { name: 'Save' }),
    page.getByRole('button', { name: 'Cancel' }),
  ];
  const bottom = (box: { y: number; height: number } | null) => (box?.y ?? 0) + (box?.height ?? 0);

  // From 320 px up to the phones' widths, at which Save alone would fit beside the priority.
  for (const width of [320, 402, 440]) {
    await page.setViewportSize({ width, height: 800 });
    const [field, priority, save, cancel] = await Promise.all(controls.map((control) => control.boundingBox()));
    expect(priority?.y, `the priority at ${width} px`).toBe(bottom(field) + 12);
    expect(priority?.x, `the priority at ${width} px`).toBe(field?.x);
    expect(priority?.height, `the priority's lines at ${width} px`).toBe(44);
    expect(save?.y, `Save at ${width} px`).toBe(bottom(priority) + 12);
    expect(cancel?.y, `Cancel at ${width} px`).toBe(save?.y);
    const scroll = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth
    );
    expect(scroll, `sideways scroll at ${width} px`).toBe(0);
  }
  await page.setViewportSize({ width: 1024, height: 768 });

  const [field, priority, save, cancel] = await Promise.all(controls.map((control) => control.boundingBox()));
  expect(priority?.y).toBe(field?.y);
  expect(save?.y).toBe(field?.y);
  expect(cancel?.y).toBe(field?.y);
});
