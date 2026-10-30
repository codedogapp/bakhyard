import type { Page } from '@playwright/test';

import type { Plan } from '../src/planner/api';
import { home, plans } from './demo';
import { expect, test } from './fixtures';
import { group } from './groups';
import { serveConversation } from './plan';

const [plan] = plans as [Plan, ...Plan[]];

/** The reply's status line. */
function replyStatus(page: Page) {
  return group(page, 'The conversation').locator('form').getByRole('status');
}

/** Answers each send to the plan with the answer once answered resolves, and returns the bodies sent. */
async function serveSends(page: Page, answer: { status: number; body?: string }, answered = Promise.resolve()) {
  const sends: unknown[] = [];
  await page.route(`/api/planner/plans/${plan.id}/messages`, async (route) => {
    sends.push(route.request().postDataJSON());
    await answered;
    await route.fulfill(answer);
  });
  return sends;
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await serveConversation(page, plan.id);
});

test("a reply goes to the plan as typed, once, and its field empties once it's sent", async ({ page }) => {
  const answered = Promise.withResolvers<void>();
  const sends = await serveSends(page, { status: 204 }, answered.promise);
  await page.goto(`/planner/${plan.id}`);
  const reply = page.getByRole('textbox', { name: 'Reply' });
  const send = page.getByRole('button', { name: 'Send' });

  await reply.fill('Make it a week.\nKeep it cheap.');
  // The second click comes while the first one's send runs.
  await send.dblclick();

  await expect(replyStatus(page)).toHaveText('Sending…');
  await expect(reply).toHaveAttribute('readonly');
  await expect(send).toHaveAttribute('aria-disabled', 'true');
  answered.resolve();
  await expect(replyStatus(page)).toHaveText('Sent');
  await expect(reply).toHaveValue('');
  await expect(reply).not.toHaveAttribute('readonly');
  expect(sends).toEqual([{ text: 'Make it a week.\nKeep it cheap.' }]);
});

test('a failed reply keeps its text and says why', async ({ page }) => {
  const sends = await serveSends(page, { status: 502, body: 'Bad Gateway\n' });
  await page.goto(`/planner/${plan.id}`);
  const reply = page.getByRole('textbox', { name: 'Reply' });

  await reply.fill('Make it a week.');
  await page.getByRole('button', { name: 'Send' }).click();

  await expect(replyStatus(page)).toHaveText("Copilot couldn't take the message.");
  await expect(reply).toHaveValue('Make it a week.');
  expect(sends).toHaveLength(1);
});
