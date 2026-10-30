import { afterEach, expect, test, vi } from 'vitest';

import {
  coverImage,
  foundGames,
  loadGames,
  platformsLabel,
  playtimeLabel,
  type Snapshot,
  trophiesLabel,
} from './games';

afterEach(() => {
  vi.unstubAllGlobals();
});

const game = { title: 'Moon Garden', platforms: ['PS5'], lastPlayed: '2026-10-30T11:00:00Z' };

test('loads the games I played lately from the games service', async () => {
  const snapshot: Snapshot = { games: [game], searched: '2026-10-30T12:00:00Z' };
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(Response.json(snapshot));
  vi.stubGlobal('fetch', fetchMock);

  expect(await loadGames()).toEqual({ kind: 'loaded', snapshot });
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/games/recent');
});

test.each<{ name: string; snapshot: Snapshot; want?: Snapshot['games'] }>([
  {
    name: 'the games of a fetch that succeeded',
    snapshot: { games: [game], searched: '2026-10-30T12:00:00Z' },
    want: [game],
  },
  {
    name: 'none of a fetch that succeeded but found none',
    snapshot: { games: [], searched: '2026-10-30T12:00:00Z', failure: 'list games: 403 Forbidden' },
    want: [],
  },
  { name: 'nothing before a fetch has succeeded', snapshot: { games: [], failure: 'list games: 403 Forbidden' } },
])('finds $name', ({ snapshot, want }) => {
  expect(foundGames(snapshot)).toEqual(want);
});

test.each([
  { cover: undefined, want: undefined },
  {
    cover: 'https://images.igdb.com/igdb/image/upload/t_cover_big/co1abc.jpg',
    want: 'url("https://images.igdb.com/igdb/image/upload/t_cover_big/co1abc.jpg")',
  },
  // A quote or a backslash would end or bend the CSS string, so they're escaped.
  { cover: 'https://example.com/a"b\\c.jpg', want: 'url("https://example.com/a\\"b\\\\c.jpg")' },
])('draws the cover $cover as the background image $want', ({ cover, want }) => {
  expect(coverImage(cover)).toBe(want);
});

test.each([
  { platforms: ['PS5'], want: 'PS5' },
  { platforms: ['PS4', 'PS5'], want: 'PS4 · PS5' },
  { platforms: [], want: '' },
])('lists the platforms $platforms as "$want"', ({ platforms, want }) => {
  expect(platformsLabel(platforms)).toBe(want);
});

test.each([
  { trophies: { earned: 12, total: 40, percent: 30 }, want: '12 of 40 trophies · 30%' },
  { trophies: { earned: 0, total: 1, percent: 0 }, want: '0 of 1 trophy · 0%' },
  { trophies: { earned: 1, total: 3, percent: 33.33 }, want: '1 of 3 trophies · 33%' },
  { trophies: { earned: 49, total: 50, percent: 99.9 }, want: '49 of 50 trophies · 99%' },
  { trophies: { earned: 50, total: 50, percent: 100 }, want: '50 of 50 trophies · 100%' },
  { trophies: { earned: 1234, total: 5678, percent: 21.7 }, want: '1,234 of 5,678 trophies · 21%' },
])('counts the trophies $trophies as "$want"', ({ trophies, want }) => {
  expect(trophiesLabel(trophies)).toBe(want);
});

test.each([
  { minutes: 750, want: '12h 30m' },
  { minutes: 45, want: '45m' },
  { minutes: 180, want: '3h' },
  { minutes: 1, want: '1m' },
  { minutes: 74_045, want: '1,234h 5m' },
])('says $minutes minutes of play are "$want"', ({ minutes, want }) => {
  expect(playtimeLabel(minutes)).toBe(want);
});
