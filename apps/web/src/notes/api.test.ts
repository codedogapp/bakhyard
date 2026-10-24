import { afterEach, expect, test, vi } from 'vitest';

import { fetchNotes, syncNotes } from './api';

afterEach(() => {
  vi.unstubAllGlobals();
});

const files = ['Daily/To do.md', 'Work/Plan.md', 'Work/diagram.png'];

test.each([
  { name: 'lists', call: fetchNotes, url: '/api/notes', method: 'GET' },
  { name: 'syncs', call: syncNotes, url: '/api/notes/sync', method: 'POST' },
])('$name the vault through the gateway', async ({ call, url, method }) => {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(Response.json({ files }));
  vi.stubGlobal('fetch', fetchMock);

  await expect(call()).resolves.toEqual(files);
  expect(fetchMock).toHaveBeenCalledWith(url, { method });
});

test.each([
  { call: fetchNotes, status: 401, want: 'GET /api/notes: 401' },
  { call: fetchNotes, status: 502, want: 'GET /api/notes: 502' },
  { call: syncNotes, status: 500, want: 'POST /api/notes/sync: 500' },
])('fails with "$want" when the gateway answers with an error', async ({ call, status, want }) => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status })));

  await expect(call()).rejects.toThrow(want);
});

test('fails when the gateway can not be reached', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));

  await expect(syncNotes()).rejects.toThrow('Failed to fetch');
});
