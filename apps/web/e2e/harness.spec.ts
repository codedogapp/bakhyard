import type { BrowserContext } from '@playwright/test';

import { home } from './demo';
import { expect, test, watch } from './fixtures';

// What every test fails on (fixtures.ts). Each test but the first makes it happen on a page of its own, which the
// fixture doesn't check, so only the expected problem shows and the test itself passes.

async function openHome(context: BrowserContext) {
  const page = await context.newPage();
  const problems = await watch(page);
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.goto('/');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Good morning, Alex.');
  return { page, problems };
}

test('a problem in the page fails the test', async ({ page }) => {
  test.fail();
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.goto('/');
  // Nothing here checks the problem, so only the fixture can fail the test.
  await Promise.all([page.waitForEvent('console'), page.evaluate(() => console.error('Something broke.'))]);
});

test('an API call that the test does not answer is a problem', async ({ context, baseURL }) => {
  const page = await context.newPage();
  const problems = await watch(page);
  await page.goto('/');
  await expect.poll(() => problems).toContain(`unanswered: GET ${baseURL}/api/home`);
});

test('a console error is a problem', async ({ context }) => {
  const { page, problems } = await openHome(context);
  await page.evaluate(() => console.error('Something broke.'));
  await expect.poll(() => problems).toContain('console: Something broke.');
});

test('an uncaught error is a problem', async ({ context }) => {
  const { page, problems } = await openHome(context);
  // Not in a timer: Playwright's clock runs the page's timers and logs their errors to the console instead.
  await page.evaluate(() => {
    const target = new EventTarget();
    target.addEventListener('break', () => {
      throw new Error('Something broke.');
    });
    target.dispatchEvent(new Event('break'));
  });
  await expect.poll(() => problems).toContain('page error: Something broke.');
});

test('an error in a timer is a problem', async ({ context }) => {
  const { page, problems } = await openHome(context);
  await page.evaluate(() => {
    setTimeout(() => {
      throw new Error('Something broke.');
    });
  });
  // Chromium adds the stack.
  await expect.poll(() => problems).toContainEqual(expect.stringMatching(/^console: Error: Something broke\.$/m));
});

test('a Content Security Policy violation is a problem', async ({ context }) => {
  const { page, problems } = await openHome(context);
  await page.evaluate(() => {
    const script = document.createElement('script');
    script.textContent = 'void 0';
    document.head.append(script);
  });
  await expect.poll(() => problems).toContain('console: CSP violation: script-src-elem inline');
});

test('a missing file is a problem', async ({ context }) => {
  const { page, problems } = await openHome(context);
  await page.evaluate(() => {
    const image = document.createElement('img');
    image.src = '/assets/missing.png';
    document.body.append(image);
  });
  await expect
    .poll(() => problems)
    .toContain('console: Failed to load resource: the server responded with a status of 404 (Not Found)');
});
