import { home } from './demo';
import { expect, test } from './fixtures';

const history = 'https://git.example.com/me/vault/commits/branch/main';
const riff = '# Riff #1\n\nPlay it **slow**.\n';

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.route('/api/notes', (route) =>
    route.fulfill({ json: { files: ['Inbox.md', 'Songs/Riff #1.md'], editable: true, history } })
  );
  await page.route('/api/notes/files/Songs/Riff%20%231.md', (route) =>
    route.fulfill({ body: riff, contentType: 'text/markdown; charset=utf-8', headers: { ETag: '"1"' } })
  );
});

test('History takes me to the note’s history in Gitea, each name in its path escaped', async ({ page }) => {
  await page.route(`${history}/**`, (route) =>
    route.fulfill({ contentType: 'text/html', body: '<title>History of Riff #1</title>' })
  );
  await page.goto('/notes/Songs/Riff%20%231');
  await expect(page.getByRole('article')).toContainText('Play it slow.');

  await page.getByRole('link', { name: 'History' }).click();

  await expect(page).toHaveURL(`${history}/Songs/Riff%20%231.md`);
  await expect(page).toHaveTitle('History of Riff #1');
});

test('History hides while the editor is open, and comes back with the note', async ({ page }) => {
  await page.goto('/notes/Songs/Riff%20%231');
  const link = page.getByRole('link', { name: 'History' });
  await expect(link).toBeVisible();

  await page.getByRole('button', { name: 'Edit' }).click();

  await expect(page.getByRole('textbox', { name: 'Markdown' })).toBeVisible();
  await expect(link).toHaveCount(0);
  await page.getByRole('button', { name: 'Cancel' }).click();
  await expect(link).toBeVisible();
});
