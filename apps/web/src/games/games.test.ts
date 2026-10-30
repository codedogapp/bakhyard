import { afterEach, expect, test, vi } from 'vitest';

import {
  coverImage,
  foundGames,
  loadGames,
  platformsLabel,
  playedLabel,
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

// The tests run in Tokyo, 9 hours ahead of UTC, where 31 October begins at 2026-10-30T15:00:00Z.
test.each([
  { lastPlayed: '2026-10-30T15:00:00Z', want: 'Today' },
  { lastPlayed: '2026-10-31T00:40:00Z', want: 'Today' },
  { lastPlayed: '2026-10-30T14:59:59Z', want: 'Yesterday' },
  { lastPlayed: '2026-10-29T15:00:00Z', want: 'Yesterday' },
  { lastPlayed: '2026-10-29T14:59:59Z', want: '29 Oct' },
  { lastPlayed: '2026-10-17T03:00:00Z', want: '17 Oct' },
  { lastPlayed: '2026-11-01T01:00:00Z', want: '1 Nov' },
])('says I last played a game at $lastPlayed on "$want"', ({ lastPlayed, want }) => {
  expect(playedLabel(lastPlayed, new Date('2026-10-31T09:41:00+09:00'))).toBe(want);
});

test('names the day across the turn of a month and a year', () => {
  expect(playedLabel('2026-12-31T03:00:00Z', new Date('2027-01-01T09:00:00+09:00'))).toBe('Yesterday');
  expect(playedLabel('2026-12-30T03:00:00Z', new Date('2027-01-01T09:00:00+09:00'))).toBe('30 Dec');
  // At 05:00 on New Year's Day in Tokyo, while UTC is still on the old year's last day.
  expect(playedLabel('2026-12-31T20:00:00Z', new Date('2027-01-01T10:00:00+09:00'))).toBe('Today');
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
