import type { Page } from '@playwright/test';

import type { Plan } from '../src/planner/api';
import { home, plans } from './demo';
import { expect, test } from './fixtures';
import { conversation, stream, themePlan } from './plan';

const [plan] = plans as [Plan, ...Plan[]];
const events = `/api/planner/plans/${plan.id}/events`;
// Long enough that the browser doesn't open a stream again by itself while a test runs.
const never = 600_000;

/** An answer that opens the stream with the events. */
function opened(retryMs: number, ...sent: [name: string, data: unknown][]) {
  return { contentType: 'text/event-stream', body: stream(retryMs, ...sent) };
}

/** Returns the URLs of the chunks that the page loads to render a plan, as it loads them. */
function renderers(page: Page): string[] {
  const loaded: string[] = [];
  page.on('request', (request) => {
    if (/\/assets\/render-[\w-]+\.js$/.test(new URL(request.url()).pathname)) {
      loaded.push(request.url());
    }
  });
  return loaded;
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test('the plan shows above the conversation once Copilot brings it, rendered, and not before', async ({ page }) => {
  const loaded = renderers(page);
  const released = Promise.withResolvers<void>();
  let opens = 0;
  // The stream opens again at once, and brings the plan once released.
  await page.route(events, async (route) => {
    opens += 1;
    if (opens === 1) {
      await route.fulfill(opened(0, ['conversation', conversation]));
      return;
    }
    await released.promise;
    await route.fulfill(opened(never, ['conversation', conversation], ['plan', { plan: themePlan }]));
  });
  await page.goto(`/planner/${plan.id}`);

  await expect(page.getByRole('heading')).toHaveText(['Plan', 'The conversation']);
  expect(loaded).toEqual([]);
  released.resolve();

  const article = page.getByRole('article');
  await expect(page.getByRole('heading')).toHaveText([
    'Plan',
    'Dark theme for the docs site',
    'Approach',
    'Steps',
    'Files',
    'The conversation',
  ]);
  await expect(article.getByRole('heading', { level: 2 })).toHaveText('Dark theme for the docs site');
  await expect(article.getByRole('checkbox')).toHaveCount(4);
  await expect(article.getByRole('row')).toHaveCount(3);
  expect(loaded).toHaveLength(1);
  const text = await article.locator(':scope > div').boundingBox();
  expect(text?.width).toBeLessThanOrEqual(700);
});

test('a page that opens after the plan came shows it, and wide blocks scroll on their own on a narrow screen', async ({
  page,
}) => {
  await page.setViewportSize({ width: 320, height: 640 });
  const columns = 12;
  const wide = [
    themePlan,
    '',
    'y'.repeat(200),
    '',
    '```',
    'x'.repeat(200),
    '```',
    '',
    `|${' Column |'.repeat(columns)}`,
    `|${' --- |'.repeat(columns)}`,
    `|${' cell |'.repeat(columns)}`,
  ].join('\n');
  const talk = { ...conversation, plan: wide, working: false };
  await page.route(events, (route) => route.fulfill(opened(never, ['conversation', talk])));
  await page.goto(`/planner/${plan.id}`);
  const article = page.getByRole('article');

  await expect(article.getByRole('heading').first()).toHaveText('Dark theme for the docs site');
  for (const block of [article.locator('pre'), article.getByRole('table').last()]) {
    expect(await block.evaluate((element) => element.scrollWidth > element.clientWidth)).toBe(true);
  }
  const scroll = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(scroll).toBe(0);
});
