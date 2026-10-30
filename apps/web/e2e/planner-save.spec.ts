import type { Page } from '@playwright/test';

import type { Plan } from '../src/planner/api';
import { home, plans } from './demo';
import { expect, test } from './fixtures';
import { conversation, serveConversation, stream, themePlan } from './plan';

const [plan] = plans as [Plan, ...Plan[]];

/** The save's status line: the page's first, above the plan. */
function saveStatus(page: Page) {
  return page.getByRole('main').getByRole('status').first();
}

/** Answers the plan's stream with the conversation and the plan Copilot wrote. */
async function servePlan(page: Page) {
  const talk = { ...conversation, plan: themePlan };
  await page.route(`/api/planner/plans/${plan.id}/events`, (route) =>
    route.fulfill({ contentType: 'text/event-stream', body: stream(600_000, ['conversation', talk]) })
  );
}

type Answer = { status: number; json?: unknown; body?: string };

/** Answers the plan's saves with the answers in turn, each once answered resolves, and returns the saves' methods. */
async function serveSaves(page: Page, answers: Answer[], answered?: Promise<void>) {
  const saves: string[] = [];
  await page.route(`/api/planner/plans/${plan.id}/note`, async (route) => {
    const answer = answers[saves.length];
    saves.push(route.request().method());
    await answered;
    await route.fulfill(answer);
  });
  return saves;
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test('the plan saves in the vault once a click, and the page says as which note', async ({ page }) => {
  await servePlan(page);
  const answered = Promise.withResolvers<void>();
  const path = 'Plans/2026-10-21-dark-theme-for-the-docs-site.md';
  const saves = await serveSaves(page, [{ status: 201, json: { path } }], answered.promise);
  await page.goto(`/planner/${plan.id}`);
  const save = page.getByRole('button', { name: 'Save to Obsidian' });

  // The second click comes while the first one's save runs.
  await save.dblclick();

  await expect(saveStatus(page)).toHaveText('Saving…');
  await expect(save).toHaveAttribute('aria-disabled', 'true');
  answered.resolve();
  await expect(saveStatus(page)).toHaveText('Saved as “Plans/2026-10-21-dark-theme-for-the-docs-site”');
  await expect(save).toHaveAttribute('aria-disabled', 'false');
  expect(saves).toEqual(['POST']);
});

test('a failed save says why, and another click tries again', async ({ page }) => {
  await servePlan(page);
  const saves = await serveSaves(page, [
    { status: 502, body: 'Bad Gateway\n' },
    { status: 404, body: 'Saving to the vault is not set up\n' },
  ]);
  await page.goto(`/planner/${plan.id}`);
  const save = page.getByRole('button', { name: 'Save to Obsidian' });

  await save.click();
  await expect(saveStatus(page)).toHaveText("Couldn't save the plan in the vault.");
  await save.click();

  await expect(saveStatus(page)).toHaveText('Saving to the vault is not set up');
  expect(saves).toEqual(['POST', 'POST']);
});

test('there is nothing to save till Copilot writes the plan', async ({ page }) => {
  await serveConversation(page, plan.id);
  await page.goto(`/planner/${plan.id}`);

  await expect(page.getByRole('heading')).toHaveText(['Plan', 'The conversation']);
  await expect(page.getByRole('button', { name: 'Save to Obsidian' })).toHaveCount(0);
});

test("a note's long name wraps on a narrow screen", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 640 });
  await servePlan(page);
  // A slug's longest word: its first 60 characters.
  const word = 'x'.repeat(60);
  await serveSaves(page, [{ status: 201, json: { path: `Plans/2026-10-21-${word}.md` } }]);
  await page.goto(`/planner/${plan.id}`);

  await page.getByRole('button', { name: 'Save to Obsidian' }).click();

  await expect(saveStatus(page)).toHaveText(`Saved as “Plans/2026-10-21-${word}”`);
  const scroll = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(scroll).toBe(0);
});
