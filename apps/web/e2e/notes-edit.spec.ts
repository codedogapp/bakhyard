import { home } from './demo';
import { expect, test } from './fixtures';

const files = ['Inbox.md', 'Songs/Riff.md'];
const riff = '---\ntags: [guitar]\n---\n# Riff\n\nPlay it **slow**.\n';

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.route('/api/notes/files/Songs/Riff.md', (route) =>
    route.fulfill({ body: riff, contentType: 'text/markdown; charset=utf-8' })
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
