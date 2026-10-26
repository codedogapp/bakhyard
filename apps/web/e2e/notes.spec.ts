import { home } from './demo';
import { expect, test } from './fixtures';

const files = ['Inbox.md', 'Maths/Euler.md', 'Maths/Primes.md'];
const euler = "# Euler\n\nEuler's identity: $e^{i\\pi} + 1 = 0$\n";

test('the Notes filter finds a note, and its page renders the math', async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.route('/api/notes', (route) => route.fulfill({ json: { files } }));
  await page.route('/api/notes/files/Maths/Euler.md', (route) =>
    route.fulfill({ body: euler, contentType: 'text/markdown; charset=utf-8' })
  );
  await page.goto('/notes');

  await expect(page.getByText('3 notes in the vault.')).toBeVisible();
  await page.getByRole('searchbox', { name: 'Filter notes' }).fill('eul');
  await page.getByRole('link', { name: /Euler/ }).click();

  await expect(page).toHaveURL('/notes/Maths/Euler');
  await expect(page).toHaveTitle('Euler · Bakhyard');
  await expect(page.locator('.katex-mathml annotation')).toHaveText('e^{i\\pi} + 1 = 0');
});
