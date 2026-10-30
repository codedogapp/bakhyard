import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import { NotesState } from './state.svelte';

const listed = ['Inbox.md', 'Work/Plan.md'];
const synced = ['Inbox.md', 'Work/Plan.md', 'Work/New.md'];

let fetchMock: ReturnType<typeof vi.fn<typeof fetch>>;

beforeEach(() => {
  fetchMock = vi.fn<typeof fetch>();
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

test('loads the list the first time only', async () => {
  fetchMock.mockResolvedValue(Response.json({ files: listed }));
  const notes = new NotesState();
  expect(notes.status).toEqual({ kind: 'idle' });

  await notes.load();
  await notes.load();

  expect(notes.files).toEqual(listed);
  expect(notes.status).toEqual({ kind: 'loaded' });
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/notes', { method: 'GET' });
});

test('tries again after a load that failed', async () => {
  fetchMock.mockResolvedValueOnce(new Response(null, { status: 502 }));
  fetchMock.mockResolvedValueOnce(Response.json({ files: listed }));
  const notes = new NotesState();

  await notes.load();

  expect(notes.files).toBeUndefined();
  expect(notes.status).toEqual({ kind: 'load-failed' });

  await notes.load();

  expect(notes.files).toEqual(listed);
  expect(notes.status).toEqual({ kind: 'loaded' });
});

test('syncs the vault and lists it as it is now', async () => {
  fetchMock.mockResolvedValueOnce(Response.json({ files: listed }));
  fetchMock.mockResolvedValueOnce(Response.json({ files: synced }));
  const notes = new NotesState();
  await notes.load();
  const before = Date.now();

  await notes.sync();

  expect(notes.files).toEqual(synced);
  expect(notes.status).toEqual({ kind: 'synced', at: expect.any(Date) });
  expect(notes.status.kind === 'synced' && notes.status.at.getTime()).toBeGreaterThanOrEqual(before);
  expect(fetchMock).toHaveBeenLastCalledWith('/api/notes/sync', { method: 'POST' });
});

test('keeps the list when a sync fails', async () => {
  fetchMock.mockResolvedValueOnce(Response.json({ files: listed }));
  fetchMock.mockRejectedValueOnce(new TypeError('Failed to fetch'));
  const notes = new NotesState();
  await notes.load();

  await notes.sync();

  expect(notes.files).toEqual(listed);
  expect(notes.status).toEqual({ kind: 'sync-failed' });
});

test('takes whether notes save from each list that loads, and keeps it when a sync fails', async () => {
  fetchMock.mockResolvedValueOnce(Response.json({ files: listed, editable: true }));
  fetchMock.mockRejectedValueOnce(new TypeError('Failed to fetch'));
  fetchMock.mockResolvedValueOnce(Response.json({ files: synced }));
  const notes = new NotesState();
  expect(notes.editable).toBe(false);

  await notes.load();
  expect(notes.editable).toBe(true);

  await notes.sync();
  expect(notes.editable).toBe(true);

  await notes.sync();
  expect(notes.editable).toBe(false);
});

test('lists the vault on a sync after a failed load, and loads nothing more', async () => {
  fetchMock.mockResolvedValueOnce(new Response(null, { status: 502 }));
  fetchMock.mockResolvedValueOnce(Response.json({ files: synced }));
  const notes = new NotesState();
  await notes.load();

  await notes.sync();
  await notes.load();

  expect(notes.files).toEqual(synced);
  expect(notes.status.kind).toBe('synced');
  expect(fetchMock).toHaveBeenCalledTimes(2);
});

test.each([
  { name: 'a load', start: (notes: NotesState) => notes.load(), kind: 'loading' },
  { name: 'a sync', start: (notes: NotesState) => notes.sync(), kind: 'syncing' },
])('starts nothing else while $name runs, and a load waits for it', async ({ start, kind }) => {
  const answer = Promise.withResolvers<Response>();
  fetchMock.mockReturnValueOnce(answer.promise);
  const notes = new NotesState();

  const running = start(notes);
  let waited = false;
  const waiting = notes.load().then(() => {
    waited = true;
  });
  await notes.sync();

  expect(notes.status.kind).toBe(kind);
  expect(notes.busy).toBe(true);
  expect(waited).toBe(false);
  expect(fetchMock).toHaveBeenCalledOnce();

  answer.resolve(Response.json({ files: listed }));
  await waiting;

  expect(notes.busy).toBe(false);
  expect(notes.files).toEqual(listed);
  expect(fetchMock).toHaveBeenCalledOnce();
  await running;
});
