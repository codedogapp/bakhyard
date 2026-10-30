import { afterEach, expect, test, vi } from 'vitest';

import { createNote, fetchNote, fetchNotes, fileUrl, historyUrl, saveNote, syncNotes } from './api';

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

const history = 'https://git.example.com/me/vault/commits/branch/main';

test('takes the vault’s history from the list, when notes has one', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(Response.json({ files, history })));

  await expect(syncNotes()).resolves.toStrictEqual({ files, editable: false, history });
});

test('gives a note’s history under the vault’s, each name in its path escaped', () => {
  expect(historyUrl(history, 'Daily/To do #1?%é.md')).toBe(`${history}/Daily/To%20do%20%231%3F%25%C3%A9.md`);
});

const version = '"9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"';

test('loads a note and its version through the gateway, each name in its path escaped', async () => {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response('# To do\n', { headers: { ETag: version } }));
  vi.stubGlobal('fetch', fetchMock);

  await expect(fetchNote('Daily/To do #1?.md')).resolves.toEqual({ markdown: '# To do\n', version });
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/notes/files/Daily/To%20do%20%231%3F.md');
});

test('gives a blank version for a note answered without one', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response('# To do\n')));

  await expect(fetchNote('Daily/To do.md')).resolves.toEqual({ markdown: '# To do\n', version: '' });
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

test('saves a note through the gateway, if it is still at the version it loaded, and gives its new version', async () => {
  const saved = '"60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752"';
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockResolvedValue(new Response(null, { status: 204, headers: { ETag: saved } }));
  vi.stubGlobal('fetch', fetchMock);

  await expect(saveNote('Daily/To do #1?.md', '# To do\n\n- [ ] Tune\n', version)).resolves.toBe(saved);
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/notes/files/Daily/To%20do%20%231%3F.md', {
    method: 'PUT',
    headers: { 'Content-Type': 'text/markdown; charset=utf-8', 'If-Match': version },
    body: '# To do\n\n- [ ] Tune\n',
  });
});

test('gives a blank version for a save answered without one', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 204 })));

  await expect(saveNote('Daily/To do.md', 'Tune', version)).resolves.toBe('');
});

test('gives no version when the note has changed since it loaded', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 412 })));

  await expect(saveNote('Daily/To do.md', 'Tune', version)).resolves.toBeUndefined();
});

test.each([400, 428, 500, 502])('fails when the gateway answers a save with %i', async (status) => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status })));

  await expect(saveNote('Daily/To do.md', 'Tune', version)).rejects.toThrow(
    `PUT /api/notes/files/Daily/To%20do.md: ${status}`
  );
});

test('fails to save a note when the gateway can not be reached', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));

  await expect(saveNote('Daily/To do.md', 'Tune', version)).rejects.toThrow('Failed to fetch');
});

test('makes a new note through the gateway, each name in its path escaped, and gives the path it got', async () => {
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockResolvedValue(Response.json({ path: 'Daily/To do #1? 1.md' }, { status: 201 }));
  vi.stubGlobal('fetch', fetchMock);

  await expect(createNote('Daily/To do #1?.md', '# To do\n\n- [ ] Tune\n')).resolves.toBe('Daily/To do #1? 1.md');
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/notes/files/Daily/To%20do%20%231%3F.md', {
    method: 'POST',
    headers: { 'Content-Type': 'text/markdown; charset=utf-8' },
    body: '# To do\n\n- [ ] Tune\n',
  });
});

test.each([400, 404, 409, 413, 502])('fails when the gateway answers a new note with %i', async (status) => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status })));

  await expect(createNote('Daily/To do.md', 'Tune')).rejects.toThrow(
    `POST /api/notes/files/Daily/To%20do.md: ${status}`
  );
});

test('fails to make a new note when the gateway can not be reached', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));

  await expect(createNote('Daily/To do.md', 'Tune')).rejects.toThrow('Failed to fetch');
});
