import type { Page } from '@playwright/test';

import type { State } from '../src/nudge/nudge';
import { home, nudge } from './demo';
import { expect, test } from './fixtures';

/**
 * What the page's calls to nudge get: the state it is given when it asks, and the answers it is sent. An answered
 * question leaves the state, as it does in nudge.
 */
async function serve(page: Page, initial: State) {
  const served = { state: initial, answers: [] as { id: string; body: unknown }[] };
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.route('/api/nudge/state', (route) => route.fulfill({ json: served.state }));
  // The stream says that it opens and asks the browser to wait ten minutes before it opens again.
  await page.route('/api/nudge/events', (route) =>
    route.fulfill({ contentType: 'text/event-stream', body: 'retry: 600000\n: open\n\n' })
  );
  await page.route('/api/nudge/questions/*/answer', (route) => {
    const id = route.request().url().split('/').at(-2) ?? '';
    served.answers.push({ id, body: route.request().postDataJSON() });
    served.state = { ...served.state, questions: served.state.questions.filter((question) => question.id !== id) };
    return route.fulfill({ status: 204 });
  });
  return served;
}

test('the Nudge page lists the questions that wait, and the agents at work', async ({ page }) => {
  await serve(page, nudge);
  await page.goto('/nudge');

  await expect(page).toHaveTitle('Nudge · Bakhyard');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Nudge');
  await expect(page.getByRole('heading', { level: 3 })).toHaveText([
    'Merge the pull request?',
    'What should the weekly review be called?',
  ]);
  await expect(page.getByText('CI is green and two reviewers approved it.')).toBeVisible();
  await expect(page.getByText('2 waiting')).toBeVisible();
  await expect(page.getByText('Drafting the review template')).toBeVisible();
});

test('a tap on an option answers the question at once', async ({ page }) => {
  const served = await serve(page, nudge);
  await page.goto('/nudge');
  await page.getByRole('button', { name: 'Yes' }).click();

  await expect(page.getByRole('heading', { level: 3 })).toHaveText(['What should the weekly review be called?']);
  expect(served.answers).toEqual([{ id: nudge.questions[0].id, body: { option: 'Yes' } }]);
});

test('a typed reply answers a question that has no options', async ({ page }) => {
  const served = await serve(page, nudge);
  await page.goto('/nudge');
  const field = page.getByRole('textbox', { name: /^Reply to: What should/ });
  await field.fill('Friday wrap-up');
  await field.press('Enter');

  await expect(page.getByRole('heading', { level: 3 })).toHaveText(['Merge the pull request?']);
  expect(served.answers).toEqual([{ id: nudge.questions[1].id, body: { text: 'Friday wrap-up' } }]);
});

test('a question that arrives while the page is open shows without a reload', async ({ page }) => {
  const served = await serve(page, { ...nudge, questions: [] });
  await page.goto('/nudge');
  await expect(page.getByText('No agent is waiting for you.')).toBeVisible();

  served.state = nudge;
  // What the service worker does when a push comes in.
  await page.evaluate(() => navigator.serviceWorker.dispatchEvent(new MessageEvent('message', { data: 'push' })));

  await expect(page.getByRole('heading', { level: 3, name: 'Merge the pull request?' })).toBeVisible();
});

test('the Nudge page is under More in the tab bar, and its card links to it', async ({ page }, testInfo) => {
  // The tab bar is the phone layout.
  test.skip(testInfo.project.name !== 'phone', 'the tab bar is the phone layout');
  await serve(page, nudge);
  await page.goto('/nudge');

  await expect(page.getByRole('link', { name: 'More', exact: true })).toHaveAttribute('aria-current', 'true');
});

test("the Nudge page says when it couldn't load", async ({ page }) => {
  await serve(page, nudge);
  await page.route('/api/nudge/state', (route) => route.fulfill({ status: 502 }));
  await page.goto('/nudge');

  await expect(page.getByText("Couldn't load the questions.")).toBeVisible();
});
