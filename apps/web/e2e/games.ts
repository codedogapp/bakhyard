import type { Page } from '@playwright/test';

import type { Snapshot } from '../src/games/games';

// Where IGDB serves the covers at the size that the games service gives (decision 305).
const igdb = 'https://images.igdb.com/igdb/image/upload/t_cover_big';

/** A made-up cover at IGDB's size, a disc on a ground, since no game's real art can be in the repo. */
function art(ground: string, disc: string): string {
  return [
    '<svg xmlns="http://www.w3.org/2000/svg" width="264" height="374">',
    `<rect width="264" height="374" fill="${ground}"/><circle cx="132" cy="150" r="72" fill="${disc}"/>`,
    '</svg>',
  ].join('');
}

const arts = new Map([
  [`${igdb}/moon.jpg`, art('#22305a', '#f2d16b')],
  [`${igdb}/tower.jpg`, art('#7d2a1e', '#f4b183')],
  [`${igdb}/harbour.jpg`, art('#14504f', '#a5dbd0')],
]);

/**
 * The games a made-up user played in the last 14 days, the last played first, as the games service found them ten
 * minutes before the tests' time: one game today, one yesterday on a Nintendo console, which has no trophies, one with
 * every trophy, and one with no playtime and no cover.
 */
export const games: Snapshot = {
  games: [
    {
      title: 'Moon Garden',
      platforms: ['PS5'],
      lastPlayed: '2026-10-21T00:05:00Z',
      playtimeMinutes: 754,
      trophies: { earned: 31, total: 52, percent: 59.6 },
      cover: `${igdb}/moon.jpg`,
    },
    {
      title: 'Tower Climb',
      platforms: ['Switch'],
      lastPlayed: '2026-10-20T12:30:00Z',
      playtimeMinutes: 1290,
      cover: `${igdb}/tower.jpg`,
    },
    {
      title: 'Harbour Lights',
      platforms: ['PS4', 'PS5'],
      lastPlayed: '2026-10-17T10:00:00Z',
      playtimeMinutes: 2475,
      trophies: { earned: 40, total: 40, percent: 100 },
      cover: `${igdb}/harbour.jpg`,
    },
    {
      title: 'Sky Rally',
      platforms: ['PC'],
      lastPlayed: '2026-10-12T09:00:00Z',
      trophies: { earned: 3, total: 35, percent: 8.57 },
    },
  ],
  searched: '2026-10-21T00:31:00Z',
};

/** Answers the Games page's call for the games with the snapshot, or with the status. */
export async function answerGames(page: Page, answer: Snapshot | number) {
  await page.route('/api/games/recent', (route) =>
    typeof answer === 'number' ? route.fulfill({ status: answer }) : route.fulfill({ json: answer })
  );
}

/** Answers the browser's requests for the demo games' covers on IGDB with their made-up art, and any other with a 404. */
export async function answerCovers(page: Page) {
  await page.route(`${igdb}/*`, (route) => {
    const body = arts.get(route.request().url());
    return body ? route.fulfill({ contentType: 'image/svg+xml', body }) : route.fulfill({ status: 404 });
  });
}
