import { afterEach, expect, test, vi } from 'vitest';

import { type List, loadItems, type Snapshot } from './github';

afterEach(() => {
  vi.unstubAllGlobals();
});

test.each<{ list: List; url: string }>([
  { list: 'mine', url: '/api/github/mine' },
  { list: 'reviews', url: '/api/github/reviews' },
  { list: 'issues', url: '/api/github/issues' },
])('loads the $list list from the github service', async ({ list, url }) => {
  const snapshot: Snapshot = {
    items: [{ id: 42, title: 'Show the order history', repo: 'acme/web', draft: false }],
    searched: '2026-10-26T00:41:00Z',
  };
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(Response.json(snapshot));
  vi.stubGlobal('fetch', fetchMock);

  expect(await loadItems(list)).toEqual({ kind: 'loaded', snapshot });
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith(url);
});

test('says when GitHub is off, since the github service answers 404 without it', async () => {
  vi.stubGlobal(
    'fetch',
    vi.fn<typeof fetch>().mockResolvedValue(new Response('GitHub is not set up\n', { status: 404 }))
  );

  expect(await loadItems('issues')).toEqual({ kind: 'off' });
});
