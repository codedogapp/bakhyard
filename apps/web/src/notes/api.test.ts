import { afterEach, expect, test, vi } from 'vitest';

import { fetchNote, fetchNotes, fileUrl, syncNotes } from './api';

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

  await expect(call()).resolves.toEqual({ files, editable: false });
  expect(fetchMock).toHaveBeenCalledWith(url, { method });
});

test.each([
  { name: 'says so', editable: true, want: true },
  { name: 'says it can not', editable: false, want: false },
  { name: 'says something else', editable: 'yes', want: false },
])('takes the vault to save notes when the service $name: $want', async ({ editable, want }) => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(Response.json({ files, editable })));

  await expect(fetchNotes()).resolves.toEqual({ files, editable: want });
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

test('gives a file’s URL, each name in its path escaped', () => {
  expect(fileUrl('My art/a b%?.png')).toBe('/api/notes/files/My%20art/a%20b%25%3F.png');
});

test('loads a note through the gateway, each name in its path escaped', async () => {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response('# To do\n'));
  vi.stubGlobal('fetch', fetchMock);

  await expect(fetchNote('Daily/To do #1?.md')).resolves.toBe('# To do\n');
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/notes/files/Daily/To%20do%20%231%3F.md');
});

test('gives no note when the vault has none at the path', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response('no such file', { status: 404 })));

  await expect(fetchNote('Daily/Gone.md')).resolves.toBeUndefined();
});

test.each([401, 500, 502])('fails when the gateway answers a note with %i', async (status) => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status })));

  await expect(fetchNote('Daily/To do.md')).rejects.toThrow(`GET /api/notes/files/Daily/To%20do.md: ${status}`);
});

test('fails to load a note when the gateway can not be reached', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));

  await expect(fetchNote('Daily/To do.md')).rejects.toThrow('Failed to fetch');
});
