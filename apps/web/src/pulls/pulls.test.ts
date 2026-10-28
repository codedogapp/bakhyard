import { afterEach, expect, test, vi } from 'vitest';

import { loadPulls, pullsNote, type Snapshot } from './pulls';

afterEach(() => {
  vi.unstubAllGlobals();
});

test('loads my open pull requests from the pulls service', async () => {
  const snapshot: Snapshot = {
    pulls: [{ id: 42, title: 'Show the order history', repo: 'acme/web', draft: false }],
    searched: '2026-10-26T00:41:00Z',
  };
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(Response.json(snapshot));
  vi.stubGlobal('fetch', fetchMock);

  expect(await loadPulls()).toEqual({ kind: 'loaded', snapshot });
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/pulls/mine');
});

test.each([
  { count: 0, want: undefined },
  { count: 1, want: '1 PR' },
  { count: 2, want: '2 PRs' },
  { count: 120, want: '120 PRs' },
])('counts $count pull requests as $want', ({ count, want }) => {
  expect(pullsNote(count)).toBe(want);
});
