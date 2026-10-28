import type { Page } from '@playwright/test';

import { bounds } from './bounds';
import { home } from './demo';
import { expect, test } from './fixtures';

const reached = (app: string) => ({ app, state: 'ok' });
const unreached = (app: string) => ({ app, state: 'unavailable' });

/** Answers the Home with the demo's items and sources. */
async function answerHome(page: Page, sources?: object[]) {
  await page.route('/api/home', (route) => route.fulfill({ json: { ...home, sources } }));
}

for (const { name, sources, want } of [
  { name: 'one source it could not reach', sources: [reached('todos'), unreached('pulls')], want: '1 source' },
  { name: 'two sources it could not reach', sources: [unreached('todos'), unreached('pulls')], want: '2 sources' },
]) {
  test(`the top bar counts ${name}, and the Home still lists the rest`, async ({ page }) => {
    await answerHome(page, sources);
    await page.goto('/');

    await expect(page.getByRole('status')).toHaveText(`Online · ${want} unavailable`);
    await expect(page.getByText('3 things need you today. 1 is late.')).toBeVisible();
  });
}

for (const { name, sources } of [
  { name: 'every source reached', sources: [reached('todos'), reached('pulls')] },
  { name: 'no sources', sources: [] },
  { name: 'an answer without sources', sources: undefined },
]) {
  test(`the top bar says Online alone with ${name}`, async ({ page }) => {
    await answerHome(page, sources);
    await page.goto('/');

    await expect(page.getByRole('status')).toHaveText('Online');
  });
}

test('on a narrow screen the count neither overflows nor meets the clock', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 640 });
  await answerHome(page, [unreached('todos'), unreached('pulls')]);
  await page.goto('/');

  const status = page.getByRole('status');
  await expect(status).toHaveText('Online · 2 sources unavailable');
  const scroll = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(scroll).toBe(0);
  const [label, clock] = [await bounds(status), await bounds(page.getByText('09:41 · Tokyo'))];
  expect(label.x + label.width).toBeLessThanOrEqual(clock.x);
});
