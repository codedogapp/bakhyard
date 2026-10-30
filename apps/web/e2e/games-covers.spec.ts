import type { Page } from '@playwright/test';

import { bounds } from './bounds';
import { home } from './demo';
import { expect, test } from './fixtures';
import { answerCovers, answerGames, games } from './games';
import { group } from './groups';
import { colour } from './ink';

const played = (page: Page) => group(page, 'Recently played');

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await answerCovers(page);
  await answerGames(page, games);
});

test("each game's cover loads from IGDB into a framed blank, which a game without one keeps", async ({ page }) => {
  const loaded: string[] = [];
  page.on('requestfinished', (request) => {
    if (request.url().startsWith('https://images.igdb.com/')) {
      loaded.push(request.url());
    }
  });
  await page.goto('/games');

  const covers = played(page).locator('.cover');
  await expect(covers).toHaveCount(4);
  // The gateway's Content Security Policy lets them in: fixtures.ts fails the test on a violation.
  const wanted = games.games.flatMap((game) => game.cover ?? []);
  await expect.poll(() => loaded.toSorted()).toEqual(wanted.toSorted());
  const [fill, rule] = [await colour(page, '--fill'), await colour(page, '--rule')];
  const looks = await covers.evaluateAll((all) =>
    all.map((cover) => {
      const { backgroundColor, backgroundImage, backgroundPosition, backgroundSize, borderTopColor, borderTopWidth } =
        getComputedStyle(cover);
      return { backgroundColor, backgroundImage, backgroundPosition, backgroundSize, borderTopColor, borderTopWidth };
    })
  );
  expect(looks).toEqual(
    games.games.map((game) => ({
      backgroundColor: fill,
      backgroundImage: game.cover ? `url("${game.cover}")` : 'none',
      backgroundPosition: '50% 50%',
      backgroundSize: 'cover',
      borderTopColor: rule,
      borderTopWidth: '1px',
    }))
  );
});

test('the cover sits between the day and the title, or under the day on a phone', async ({ page }, testInfo) => {
  await page.goto('/games');

  const rows = played(page).locator('.list-row');
  await expect(rows).toHaveCount(4);
  for (const row of await rows.all()) {
    const [box, label, cover, text] = await Promise.all(
      [row, row.locator('.list-label'), row.locator('.cover'), row.locator('.list-text')].map(bounds)
    );
    expect([cover.width, cover.height]).toEqual([48, 68]);
    expect(text.x - (cover.x + cover.width)).toBe(20);
    if (testInfo.project.name === 'phone') {
      // The day shares the first line with the playtime; the cover starts the second.
      expect(cover.x - box.x).toBe(20);
      expect(cover.y - (label.y + label.height)).toBeCloseTo(6, 1);
    } else {
      // After the row's 20 px of padding, the label's 120 px column and the 20 px gap.
      expect(cover.x - box.x).toBe(160);
      expect(cover.y + cover.height / 2).toBeCloseTo(box.y + box.height / 2, 1);
    }
  }
});
