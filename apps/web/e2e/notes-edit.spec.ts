import type { Page, Request } from '@playwright/test';

import { home } from './demo';
import { expect, test } from './fixtures';

const files = ['Inbox.md', 'Songs/Riff.md'];
const riff = '---\ntags: [guitar]\n---\n# Riff\n\nPlay it **slow**.\n';

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.route('/api/notes/files/Songs/Riff.md', (route) =>
    route.fulfill({ body: riff, contentType: 'text/markdown; charset=utf-8', headers: { ETag: '"1"' } })
  );
});

test('Edit shows the note’s Markdown in the editor, which takes typing, and Cancel shows the note again', async ({
  page,
}) => {
  await page.route('/api/notes', (route) => route.fulfill({ json: { files, editable: true } }));
  await page.goto('/notes/Songs/Riff');
  await expect(page.getByRole('article')).toContainText('Play it slow.');

  await page.getByRole('button', { name: 'Edit' }).click();

  const editor = page.getByRole('textbox', { name: 'Markdown' });
  await expect(editor).toBeFocused();
  await expect(page.getByRole('article')).toHaveCount(0);
  const lines = editor.locator('.cm-line');
  await expect(lines).toHaveText(['---', 'tags: [guitar]', '---', '# Riff', '', 'Play it **slow**.', '']);
  await lines.nth(5).click();
  await page.keyboard.press('End');
  await page.keyboard.type(' Then fast.');
  await expect(lines.nth(5)).toHaveText('Play it **slow**. Then fast.');

  await page.getByRole('button', { name: 'Cancel' }).click();

  await expect(page.getByRole('article')).toContainText('Play it slow.');
  await expect(page.getByRole('article')).not.toContainText('Then fast.');
  await expect(editor).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Edit' })).toBeFocused();
});

test('a note offers no Edit while the notes service saves no note', async ({ page }) => {
  await page.route('/api/notes', (route) => route.fulfill({ json: { files } }));
  await page.goto('/notes/Songs/Riff');

  await expect(page.getByRole('article')).toContainText('Play it slow.');
  await expect(page.getByRole('button', { name: 'Edit' })).toHaveCount(0);
});

// Answers the note's saves with status, and keeps their requests.
async function saves(page: Page, status: number): Promise<Request[]> {
  const requests: Request[] = [];
  await page.route('/api/notes/files/Songs/Riff.md', (route) => {
    if (route.request().method() !== 'PUT') {
      return route.fallback();
    }
    requests.push(route.request());
    return route.fulfill({ status, headers: status === 204 ? { ETag: '"2"' } : {} });
  });
  return requests;
}

// The note's page from a notes service that saves notes, with the note in the editor and " Then fast." typed in it.
async function typed(page: Page) {
  await page.route('/api/notes', (route) => route.fulfill({ json: { files, editable: true } }));
  await page.goto('/notes/Songs/Riff');
  await page.getByRole('button', { name: 'Edit' }).click();
  const line = page.getByRole('textbox', { name: 'Markdown' }).locator('.cm-line').nth(5);
  await line.click();
  await page.keyboard.press('End');
  await page.keyboard.type(' Then fast.');
  return line;
}

test('Save sends the note with the version it loaded, then shows the note and says it saved', async ({ page }) => {
  const requests = await saves(page, 204);
  await typed(page);

  await page.getByRole('button', { name: 'Save' }).click();

  await expect(page.getByRole('article')).toContainText('Play it slow. Then fast.');
  await expect(page.getByText('Saved', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Edit' })).toBeFocused();
  expect(requests).toHaveLength(1);
  expect(requests[0]?.headers()).toMatchObject({ 'if-match': '"1"', 'content-type': 'text/markdown; charset=utf-8' });
  expect(requests[0]?.postData()).toBe(riff.replace('**slow**.', '**slow**. Then fast.'));
});

test('a note changed since it loaded keeps the editor and the text, and says so', async ({ page }) => {
  await saves(page, 412);
  const line = await typed(page);

  await page.getByRole('button', { name: 'Save' }).click();

  await expect(page.getByText('Not saved: the note has changed since it loaded')).toBeVisible();
  await expect(line).toHaveText('Play it **slow**. Then fast.');
  await expect(page.getByRole('article')).toHaveCount(0);
});

test('a changed note offers to save the text as a new note, then shows the note as it is now', async ({ page }) => {
  await saves(page, 412);
  await typed(page);
  await page.getByRole('button', { name: 'Save' }).click();
  await expect(page.getByText('Not saved: the note has changed since it loaded')).toBeVisible();
  const posts: Request[] = [];
  await page.route('/api/notes/files/Songs/Riff.md', (route) => {
    if (route.request().method() === 'POST') {
      posts.push(route.request());
      return route.fulfill({ status: 201, json: { path: 'Songs/Riff 1.md' } });
    }
    return route.fulfill({ body: riff.replace('slow', 'fast'), headers: { ETag: '"3"' } });
  });

  await page.getByRole('button', { name: 'Save as new note' }).click();

  await expect(page.getByText('Saved as “Riff 1”')).toBeVisible();
  await expect(page.getByRole('article')).toContainText('Play it fast.');
  await expect(page.getByRole('button', { name: 'Edit' })).toBeFocused();
  await expect(page.getByRole('button', { name: 'Save as new note' })).toHaveCount(0);
  expect(posts).toHaveLength(1);
  expect(posts[0]?.headers()).toMatchObject({ 'content-type': 'text/markdown; charset=utf-8' });
  expect(posts[0]?.postData()).toBe(riff.replace('**slow**.', '**slow**. Then fast.'));
});

test('⌘S, or Ctrl+S, in the editor saves', async ({ page }, testInfo) => {
  // A phone has no ⌘ or Ctrl key without a keyboard.
  test.skip(testInfo.project.name === 'phone');
  const requests = await saves(page, 204);
  await typed(page);

  await page.keyboard.press('ControlOrMeta+s');

  await expect(page.getByText('Saved', { exact: true })).toBeVisible();
  expect(requests).toHaveLength(1);
});
