import { render, screen, within } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import GamesPage from './GamesPage.svelte';
import type { Snapshot } from './games';

afterEach(() => {
  vi.unstubAllGlobals();
});

function answer(response: () => Promise<Response>) {
  const fetchMock = vi.fn<typeof fetch>(response);
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

// Saturday 31 October 2026, 09:41 in Tokyo, where the tests run.
const now = new Date('2026-10-31T09:41:00+09:00');
const searched = '2026-10-31T00:30:00Z';
const cover = 'https://images.igdb.com/igdb/image/upload/t_cover_big/co1abc.jpg';
const recent: Snapshot = {
  games: [
    {
      title: 'Moon Garden',
      platforms: ['PS4', 'PS5'],
      lastPlayed: '2026-10-31T00:20:00Z',
      playtimeMinutes: 750,
      trophies: { earned: 12, total: 40, percent: 30 },
      cover,
    },
    { title: 'Tower Climb', platforms: ['Switch'], lastPlayed: '2026-10-30T11:00:00Z', playtimeMinutes: 45 },
    { title: 'Harbour Lights', platforms: ['PC'], lastPlayed: '2026-10-19T08:00:00Z' },
  ],
  searched,
};

const loaded = (snapshot: Snapshot) => async () => Response.json(snapshot);
const rows = () => within(screen.getByRole('list')).getAllByRole('listitem');

test('lists the games I played lately as it shows, a row each, none a link', async () => {
  const fetchMock = answer(loaded(recent));
  render(GamesPage, { props: { now } });

  expect(screen.getByRole('heading', { level: 1, name: 'Games' })).toBeInTheDocument();
  expect(await screen.findByRole('heading', { level: 2, name: 'Recently played' })).toBeInTheDocument();
  expect(screen.getByText('3 games')).toBeInTheDocument();
  expect(rows().map((row) => row.textContent?.replaceAll(/\s+/g, ' ').trim())).toEqual([
    'Today Moon Garden PS4 · PS5 12 of 40 trophies · 30% 12h 30m',
    'Yesterday Tower Climb Switch 45m',
    '19 Oct Harbour Lights PC',
  ]);
  expect(within(rows()[0]).getByText('Today')).toHaveAttribute('datetime', '2026-10-31T00:20:00Z');
  expect(screen.queryByRole('link')).not.toBeInTheDocument();
  expect(screen.queryByText('Stale')).not.toBeInTheDocument();
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/games/recent');
});

test("shows each game's cover as its blank's background, which screen readers skip", async () => {
  answer(loaded(recent));
  render(GamesPage, { props: { now } });

  await screen.findByRole('list');
  const blanks = rows().map((row) => row.querySelector<HTMLElement>('.cover'));
  expect(blanks.map((blank) => blank?.style.backgroundImage)).toEqual([`url("${cover}")`, '', '']);
  expect(blanks.every((blank) => blank?.childNodes.length === 0)).toBe(true);
  expect(screen.queryByRole('img')).not.toBeInTheDocument();
});

test('marks the games stale when the latest fetch failed, and says when they are from', async () => {
  answer(loaded({ ...recent, failure: 'list games: 403 Forbidden' }));
  render(GamesPage, { props: { now } });

  expect(await screen.findByText('Stale')).toHaveClass('chip');
  expect(screen.getByText('3 games')).toContainElement(screen.getByText('Stale'));
  expect(screen.getByText(/^The last search failed/)).toHaveTextContent(
    'The last search failed: list games: 403 Forbidden. These games are from 31 October at 09:30.'
  );
  expect(rows()).toHaveLength(3);
});

test('says why the first fetch failed, with no list, count or chip', async () => {
  answer(loaded({ games: [], failure: 'list games: 403 Forbidden' }));
  render(GamesPage, { props: { now } });

  expect(await screen.findByText(/^The last search failed/)).toHaveTextContent(
    /^The last search failed: list games: 403 Forbidden\.$/
  );
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
  expect(screen.queryByText('No games played in the last 14 days.')).not.toBeInTheDocument();
  expect(screen.queryByText('Stale')).not.toBeInTheDocument();
});

test('says when I played no game in the last 14 days', async () => {
  answer(loaded({ games: [], searched }));
  render(GamesPage, { props: { now } });

  expect(await screen.findByText('No games played in the last 14 days.')).toBeInTheDocument();
  expect(screen.queryByText(/^\d+ games?$/)).not.toBeInTheDocument();
});

test("says when Exophase isn't set up, with no section", async () => {
  answer(async () => new Response('Exophase is not set up\n', { status: 404 }));
  render(GamesPage, { props: { now } });

  expect(await screen.findByText("Exophase isn't set up.")).toHaveClass('page-status');
  expect(screen.queryByRole('heading', { level: 2 })).not.toBeInTheDocument();
});

test("says when the games couldn't load", async () => {
  answer(async () => new Response('Bad Gateway\n', { status: 502 }));
  render(GamesPage, { props: { now } });

  expect(await screen.findByText("Couldn't load the games.")).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 2, name: 'Recently played' })).toBeInTheDocument();
});

test('shows the head alone, with no summary, while the games load', () => {
  answer(() => Promise.withResolvers<Response>().promise);
  const { container } = render(GamesPage, { props: { now } });

  expect(screen.getByRole('heading', { level: 1, name: 'Games' })).toBeInTheDocument();
  expect(container).toHaveTextContent(/^Games$/);
});
