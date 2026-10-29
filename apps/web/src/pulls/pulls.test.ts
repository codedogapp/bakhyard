import { afterEach, expect, test, vi } from 'vitest';

import { loadPulls, loadReviews, pullsNote, type Snapshot } from './pulls';

afterEach(() => {
  vi.unstubAllGlobals();
});

test.each([
  { name: 'my open pull requests', load: loadPulls, url: '/api/pulls/mine' },
  { name: 'the open pull requests that wait for my review', load: loadReviews, url: '/api/pulls/reviews' },
])('loads $name from the pulls service', async ({ load, url }) => {
  const snapshot: Snapshot = {
    pulls: [{ id: 42, title: 'Show the order history', repo: 'acme/web', draft: false }],
    searched: '2026-10-26T00:41:00Z',
  };
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(Response.json(snapshot));
  vi.stubGlobal('fetch', fetchMock);

  expect(await load()).toEqual({ kind: 'loaded', snapshot });
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith(url);
});

test.each([
  { count: 0, want: undefined },
  { count: 1, want: '1 PR' },
  { count: 2, want: '2 PRs' },
  { count: 120, want: '120 PRs' },
])('counts $count pull requests as $want', ({ count, want }) => {
  expect(pullsNote(count)).toBe(want);
});
