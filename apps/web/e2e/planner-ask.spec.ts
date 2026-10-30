import type { Page } from '@playwright/test';

import type { Plan } from '../src/planner/api';
import type { Question } from '../src/planner/conversation';
import { home, plans } from './demo';
import { expect, test } from './fixtures';
import { group } from './groups';
import { conversation, question, stream } from './plan';

const [plan] = plans as [Plan, ...Plan[]];
const events = `/api/planner/plans/${plan.id}/events`;
// Long enough that the browser doesn't open a stream again by itself while a test runs.
const never = 600_000;

/** An answer that opens the stream with the conversation, the question waiting, then the events. */
function asked(retryMs: number, waits: Question, ...sent: [name: string, data: unknown][]) {
  const talk = { ...conversation, questions: [waits] };
  return { contentType: 'text/event-stream', body: stream(retryMs, ['conversation', talk], ...sent) };
}

/** Answers each answer to the question with the reply once released resolves, and returns the bodies sent. */
async function serveAnswers(page: Page, reply: { status: number; body?: string }, released = Promise.resolve()) {
  const answers: unknown[] = [];
  await page.route(`/api/planner/plans/${plan.id}/questions/${question.id}/answer`, async (route) => {
    answers.push(route.request().postDataJSON());
    await released;
    await route.fulfill(reply);
  });
  return answers;
}

/** The rows of the plan's conversation. */
function rows(page: Page) {
  return group(page, 'The conversation').getByRole('listitem');
}

/** The question's row: the one with buttons. */
function asking(page: Page) {
  return rows(page).filter({ has: page.getByRole('button') });
}

/** The line that says whether Copilot works: the conversation's own status, not the question's or the reply's. */
function work(page: Page) {
  return group(page, 'The conversation').locator(':scope > [role="status"]');
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test('a choice answers the question once, and the question and its answer join the messages', async ({ page }) => {
  const released = Promise.withResolvers<void>();
  const answers = await serveAnswers(page, { status: 204 }, released.promise);
  // Once the planner has the answer, the question goes, then joins the messages with its answer (decision 412).
  const after: [name: string, data: unknown][] = [
    ['answered', question],
    ['message', { id: 'a3', from: 'assistant', text: question.text }],
    ['message', { id: 'u3', from: 'user', text: 'Follow the system' }],
  ];
  let opens = 0;
  // The stream opens again at once, and brings the rest once the planner has the answer.
  await page.route(events, async (route) => {
    opens += 1;
    if (opens === 1) {
      await route.fulfill(asked(0, question));
      return;
    }
    await released.promise;
    await route.fulfill(asked(never, question, ...after));
  });
  await page.goto(`/planner/${plan.id}`);
  const choice = asking(page).getByRole('button', { name: 'Follow the system' });

  await expect(rows(page)).toHaveCount(5);
  await expect(asking(page)).toHaveText(/^Copilot\s*Should the theme follow the system's setting/);
  await expect(asking(page).getByRole('button')).toHaveText(['Follow the system', 'Add a toggle', 'Send']);
  await expect(work(page)).toHaveText('Copilot waits for your answer.');
  await expect(page.getByRole('textbox', { name: 'Reply' })).toBeHidden();
  // The second click comes while the first one's send runs.
  await choice.dblclick();
  await expect(asking(page).getByRole('status')).toHaveText('Sending…');
  await expect(choice).toHaveAttribute('aria-disabled', 'true');
  released.resolve();

  await expect(asking(page)).toHaveCount(0);
  await expect(rows(page)).toHaveCount(6);
  await expect(rows(page).nth(4)).toHaveText(`Copilot ${question.text}`);
  await expect(rows(page).nth(5)).toHaveText('You Follow the system');
  await expect(work(page)).toHaveText('Copilot is working…');
  await expect(page.getByRole('textbox', { name: 'Reply' })).toBeVisible();
  expect(answers).toEqual([{ text: 'Follow the system' }]);
});

test('an answer of your own goes with Enter, and a failed one keeps its text and says why', async ({ page }) => {
  const answers = await serveAnswers(page, { status: 404, body: 'no such question\n' });
  await page.route(events, (route) => route.fulfill(asked(never, question)));
  await page.goto(`/planner/${plan.id}`);
  const field = page.getByRole('textbox', { name: `Answer to: ${question.text}` });
  const text = 'Both: the system by default, with a toggle in the header.';

  await expect(field).toHaveAttribute('placeholder', 'Type an answer');
  await field.fill(text);
  await field.press('Enter');

  await expect(asking(page).getByRole('status')).toHaveText('Copilot no longer waits for this question.');
  await expect(field).toHaveValue(text);
  expect(answers).toEqual([{ text }]);
});

test('a long question and long choices wrap on a narrow screen', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 640 });
  const long = {
    ...question,
    text: `Which one?\n${'x'.repeat(200)}`,
    choices: ['y'.repeat(200), 'Follow the system, and keep the light theme for those who never chose one'],
  };
  await page.route(events, (route) => route.fulfill(asked(never, long)));
  await page.goto(`/planner/${plan.id}`);

  await expect(asking(page).getByRole('button')).toHaveText([...long.choices, 'Send']);
  // A regex matches the rendered text as it is, so this checks that the line break shows.
  await expect(asking(page).getByText('Which one?')).toHaveText(/^Which one\?\nx/, { useInnerText: true });
  const scroll = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(scroll).toBe(0);
});
