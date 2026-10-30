import type { Page } from '@playwright/test';

import { bounds } from './bounds';
import { chipLooks } from './chips';
import { home } from './demo';
import { expect, test } from './fixtures';
import { answerCovers, answerGames, games } from './games';
import { group } from './groups';

const played = (page: Page) => group(page, 'Recently played');
const failure = 'list games: Get "https://www.exophase.com/public/player/1/games": 403 Forbidden';

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await answerCovers(page);
});

test('the Games page lists the games I played in the last 14 days', async ({ page }, testInfo) => {
  await answerGames(page, games);
  await page.goto('/games');

  await expect(page).toHaveTitle('Games · Bakhyard');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Games');
  // The tab bar, the phone layout, has no tab for Games.
  const [name, current] = testInfo.project.name === 'phone' ? ['More', 'true'] : ['Games', 'page'];
  await expect(page.getByRole('link', { name, exact: true })).toHaveAttribute('aria-current', current);
  await expect(played(page).getByText('4 games')).toBeVisible();
  await expect(played(page).getByRole('listitem')).toHaveText([
    'Today Moon Garden PS5 31 of 52 trophies · 59% 12h 34m',
    'Yesterday Tower Climb Switch 21h 30m',
    '17 Oct Harbour Lights PS4 · PS5 40 of 40 trophies · 100% 41h 15m',
    '12 Oct Sky Rally PC 3 of 35 trophies · 8%',
  ]);
  await expect(played(page).getByText('Today')).toHaveAttribute('datetime', '2026-10-21T00:05:00Z');
  await expect(page.getByRole('main').getByRole('link')).toHaveCount(0);
  await expect(page.getByRole('main').locator('.chip')).toHaveCount(0);
});

test('the Games page marks the games stale when the latest fetch failed', async ({ page }) => {
  await answerGames(page, { ...games, failure });
  await page.goto('/games');

  const line = played(page).getByText(/^The last search failed/);
  await expect(line).toHaveText(`The last search failed: ${failure}. These games are from 21 October at 09:31.`);
  await expect(line.locator('time')).toHaveAttribute('datetime', '2026-10-21T00:31:00Z');
  await expect(played(page).getByRole('listitem')).toHaveCount(4);
  const chip = played(page).locator('.chip');
  await expect(chip).toHaveText('Stale');
  expect(await chipLooks(chip)).toEqual(['outlined']);
  // The chip follows the count, 10 px after it and centred on its line.
  const note = played(page).getByText('4 games');
  await expect(note).toHaveText('4 games Stale');
  const count = await note.evaluate((line) => {
    const text = document.createRange();
    text.setStart(line.firstChild as Text, 0);
    text.setEnd(line.firstChild as Text, '4 games'.length);
    const { x, y, width, height } = text.getBoundingClientRect();
    return { x, y, width, height };
  });
  const badge = await bounds(chip);
  expect(badge.x - (count.x + count.width)).toBeCloseTo(10, 0);
  // Within a pixel, since the text's box comes from its font's metrics, which the browser rounds.
  expect(Math.abs(badge.y + badge.height / 2 - (count.y + count.height / 2))).toBeLessThan(1);
  const [above, card] = [await bounds(line), await bounds(played(page).locator('.card'))];
  expect(card.y - (above.y + above.height)).toBeCloseTo(20, 1);
});

test('before a fetch succeeds, the Games page says why it failed, with no list', async ({ page }) => {
  await answerGames(page, { games: [], failure });
  await page.goto('/games');

  await expect(played(page).getByText(/^The last search failed/)).toHaveText(`The last search failed: ${failure}.`);
  await expect(played(page).locator('.card')).toHaveCount(0);
  await expect(played(page).locator('.chip')).toHaveCount(0);
  await expect(played(page).getByText(/^\d+ games?$/)).toHaveCount(0);
});

test('the Games page says when I played no game in the last 14 days', async ({ page }) => {
  await answerGames(page, { games: [], searched: games.searched });
  await page.goto('/games');

  await expect(played(page).getByText('No games played in the last 14 days.')).toBeVisible();
  await expect(played(page).getByText(/^\d+ games?$/)).toHaveCount(0);
});

test('the Games page says when the games could not load', async ({ page }) => {
  await answerGames(page, 500);
  await page.goto('/games');

  await expect(played(page).getByText("Couldn't load the games.")).toBeVisible();
  await expect(played(page).getByRole('list')).toHaveCount(0);
});

test("the Games page says when Exophase isn't set up, with no section", async ({ page }) => {
  await page.route('/api/games/recent', (route) => route.fulfill({ status: 404, body: 'Exophase is not set up\n' }));
  await page.goto('/games');

  await expect(page.getByRole('main').getByText("Exophase isn't set up.")).toBeVisible();
  await expect(page.getByRole('main').getByRole('heading', { level: 2 })).toHaveCount(0);
});

test('the Games page loads the games again each time it shows', async ({ page }) => {
  let snapshot = games;
  await page.route('/api/games/recent', (route) => route.fulfill({ json: snapshot }));
  await page.goto('/games');
  await expect(played(page).getByText('4 games')).toBeVisible();

  await page.getByRole('link', { name: 'Home', exact: true }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Good morning, Alex.' })).toBeVisible();
  // The games service finds a game played while the page is away.
  const [first] = games.games;
  snapshot = { ...games, games: [{ ...first, title: 'Night Ferry' }, ...games.games] };
  await page
    .getByRole('main')
    .getByRole('link', { name: /Games Recent games and stats from Exophase/ })
    .click();

  await expect(played(page).getByText('5 games')).toBeVisible();
  await expect(played(page).getByRole('listitem').first()).toContainText('Night Ferry');
});

test('a long title, platform or failure wraps on a narrow screen', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 640 });
  const game = {
    title: `Moon ${'x'.repeat(200)}`,
    platforms: [`Console ${'y'.repeat(60)}`, 'PS5'],
    lastPlayed: '2026-10-21T00:05:00Z',
    playtimeMinutes: 74_045,
    trophies: { earned: 1234, total: 5678, percent: 21.7 },
  };
  await answerGames(page, { games: [game], searched: games.searched, failure: `${failure} ${'z'.repeat(200)}` });
  await page.goto('/games');

  await expect(played(page).getByRole('listitem')).toHaveText(
    `Today ${game.title} ${game.platforms.join(' · ')} 1,234 of 5,678 trophies · 21% 1,234h 5m`
  );
  const scroll = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(scroll).toBe(0);
});
