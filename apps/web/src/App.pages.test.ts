import { render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import App from './App.svelte';
import type { Home } from './home/api';

const home: Home = { name: 'Ada', items: [] };
const searched = '2026-10-20T00:41:00Z';

// The answer for each request that an app's page makes; the Home's for any other.
function answer(url: string): Response {
  switch (url) {
    case '/api/todos':
      return Response.json({ todos: [{ id: 1, title: 'Buy milk' }] });
    case '/api/pulls/mine':
    case '/api/pulls/reviews':
      return Response.json({ pulls: [], searched });
    case '/api/github/mine':
    case '/api/github/reviews':
    case '/api/github/issues':
      return Response.json({ items: [], searched });
    case '/api/games/recent':
      return Response.json({ games: [], searched });
    case '/api/planner/plans':
      return Response.json({ plans: [] });
    case '/api/notify/alerts':
      return Response.json({ alerts: [] });
    case '/api/time/report':
      return Response.json({
        days: [{ date: '2026-10-20', loggedHours: 7.5, missing: false }],
        week: { loggedHours: 7.5, targetHours: 40 },
        searched,
      });
    default:
      return Response.json(home);
  }
}

beforeEach(() => {
  vi.stubGlobal(
    'fetch',
    vi.fn<typeof fetch>(async (url) => answer(String(url)))
  );
});

afterEach(() => {
  vi.unstubAllGlobals();
  history.replaceState(null, '', '/');
});

test('shows the page at the path it opens on', async () => {
  history.replaceState(null, '', '/github');
  render(App);

  expect(screen.getByRole('heading', { level: 1, name: 'GitHub' })).toBeInTheDocument();
  expect(await screen.findByRole('heading', { level: 2, name: 'Your PRs' })).toBeInTheDocument();
  expect(document.title).toBe('GitHub · Bakhyard');
  expect(screen.getByRole('link', { name: 'GitHub', current: 'page' })).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'More', current: true })).toBeInTheDocument();
});

test.each([
  ['/todos', '1 todo on the list.', 'Todos · Bakhyard'],
  ['/pulls', 'No open pull requests.', 'Pull requests · Bakhyard'],
  ['/time', '7h 30m / 40h', 'Time · Bakhyard'],
  ['/github', 'No open issues.', 'GitHub · Bakhyard'],
  ['/games', 'No games played in the last 14 days.', 'Games · Bakhyard'],
  ['/planner', 'No plans yet.', 'Planner · Bakhyard'],
  ['/alerts', 'No alerts are firing.', 'Alerts · Bakhyard'],
])('shows the page of the app at %s, with what it loads', async (path, loaded, title) => {
  history.replaceState(null, '', path);
  render(App);

  expect(await screen.findByText(loaded)).toBeInTheDocument();
  expect(document.title).toBe(title);
});

test("shows a plan's page at its path, following the plan's stream", async () => {
  const opened: string[] = [];
  vi.stubGlobal(
    'EventSource',
    class {
      constructor(url: string) {
        opened.push(url);
      }
      addEventListener() {
        // The test only checks which stream the page opens.
      }
      close() {
        // Nothing to close.
      }
    }
  );
  history.replaceState(null, '', '/planner/p1');
  render(App);

  expect(await screen.findByRole('heading', { level: 1, name: 'Plan' })).toBeInTheDocument();
  expect(document.title).toBe('Plan · Bakhyard');
  expect(opened).toEqual(['/api/planner/plans/p1/events']);
});
