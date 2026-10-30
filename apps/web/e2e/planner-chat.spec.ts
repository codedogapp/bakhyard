import type { Page, Route } from '@playwright/test';

import type { Plan } from '../src/planner/api';
import { home, plans } from './demo';
import { expect, test } from './fixtures';
import { group } from './groups';
import { conversation, stream } from './plan';

const [plan] = plans as [Plan, ...Plan[]];
const events = `/api/planner/plans/${plan.id}/events`;
// Long enough that the browser doesn't open a stream again by itself while a test runs.
const never = 600_000;
const delta = (text: string) => ({ id: 'a3', from: 'assistant', text });

/** Answers the page's stream with each answer in turn, and the last one ever after. */
async function serve(page: Page, ...answers: Parameters<Route['fulfill']>[0][]) {
  await page.route(events, (route) => route.fulfill(answers.length > 1 ? answers.shift() : answers[0]));
}

/** An answer that opens the stream with the events. */
function opened(retryMs: number, ...sent: [name: string, data: unknown][]) {
  return { contentType: 'text/event-stream', body: stream(retryMs, ...sent) };
}

/** The rows of the plan's conversation. */
function rows(page: Page) {
  return group(page, 'The conversation').getByRole('listitem');
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test("a plan's page shows its conversation as it streams, and that Copilot is working", async ({ page }, testInfo) => {
  await serve(
    page,
    opened(
      never,
      ['conversation', conversation],
      ['delta', delta("I'll start with the colours")],
      ['delta', delta('.')]
    )
  );
  await page.goto(`/planner/${plan.id}`);

  await expect(page).toHaveTitle('Plan · Bakhyard');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Plan');
  // The tab bar, the phone layout, has no tab for the Planner.
  const [name, current] = testInfo.project.name === 'phone' ? ['More', 'true'] : ['Planner', 'page'];
  await expect(page.getByRole('link', { name, exact: true })).toHaveAttribute('aria-current', current);
  await expect(rows(page)).toHaveText([
    'You Add a dark theme to the docs site. Keep the light one as it is.',
    'Copilot Should the code blocks and the diagrams get dark colours too?',
    'You Yes, all of it.',
    /^Copilot Here's what I found:\s+1\. The colours/,
    "Copilot I'll start with the colours.",
  ]);
  // A regex matches the rendered text as it is, so this checks that the line break shows.
  await expect(rows(page).first()).toHaveText(/the docs site\.\nKeep the light one/, { useInnerText: true });
  await expect(group(page, 'The conversation').getByRole('status')).toHaveText('Copilot is working…');
});

test("a plan's page drops the message that Copilot didn't finish, and says why it stopped", async ({ page }) => {
  const failure = 'You have no premium requests left.';
  await serve(
    page,
    opened(
      never,
      ['conversation', conversation],
      ['delta', delta("I'll start")],
      ['status', { working: false, failure }]
    )
  );
  await page.goto(`/planner/${plan.id}`);

  await expect(group(page, 'The conversation').getByRole('status')).toHaveText(`Copilot stopped: ${failure}`);
  await expect(rows(page)).toHaveCount(4);
});

test("a plan's page says when the plan doesn't load", async ({ page }) => {
  await serve(page, { status: 404, body: 'no plan\n' });
  await page.goto(`/planner/${plan.id}`);

  await expect(page.getByText("Couldn't load the plan.")).toBeVisible();
  await expect(page.getByRole('main').getByRole('heading', { level: 2 })).toHaveCount(0);
});

test("a plan's page follows the plan again once its stream breaks, as when the planner restarts", async ({ page }) => {
  await page.clock.install();
  const answered = { ...conversation, working: false, messages: [...conversation.messages, delta('Done.')] };
  // The stream ends, the browser opens it again at once, the planner isn't back yet, and then it is.
  await serve(
    page,
    opened(0, ['conversation', conversation]),
    { status: 502 },
    opened(never, ['conversation', answered])
  );
  await page.goto(`/planner/${plan.id}`);
  await expect(rows(page)).toHaveCount(4);

  // The page waits 5 s from the 502, which may come after a first step of the clock.
  await expect(async () => {
    await page.clock.runFor(5_000);
    await expect(rows(page)).toHaveCount(5, { timeout: 500 });
  }).toPass();
  await expect(rows(page).last()).toHaveText('Copilot Done.');
  await expect(group(page, 'The conversation').getByRole('status')).toHaveText('');
});

test("a plan's long words and a long failure wrap on a narrow screen", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 640 });
  const word = 'x'.repeat(200);
  const failure = `See https://example.com/${'docs/'.repeat(30)}limits.`;
  const talk = { ...conversation, messages: [{ id: 'u1', from: 'user', text: `Plan ${word}` }] };
  await serve(page, opened(never, ['conversation', talk], ['status', { working: false, failure }]));
  await page.goto(`/planner/${plan.id}`);

  await expect(group(page, 'The conversation').getByRole('status')).toHaveText(`Copilot stopped: ${failure}`);
  await expect(rows(page)).toHaveText([`You Plan ${word}`]);
  const scroll = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(scroll).toBe(0);
});

test('a path under a plan is no page', async ({ page }) => {
  await page.goto(`/planner/${plan.id}/messages`);

  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Page not found');
});
