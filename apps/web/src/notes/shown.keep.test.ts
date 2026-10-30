import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import { renderNote } from './render';
import { ShownNote } from './shown.svelte';
import { NotesState } from './state.svelte';
import { noteAt } from './tree';

const toDo = noteAt('/notes/Daily/To%20do')!;
const url = '/api/notes/files/Daily/To%20do.md';
const files = ['Daily/To do.md', 'Inbox.md'];
const markdown = '# To do\n\n- [ ] Tune';
const ticked = '# To do\n\n- [x] Tune';
const changed = '# To do\n\n- [ ] Tune\n- [ ] Sing';
const loaded = '"1"';

let fetchMock: ReturnType<typeof vi.fn<typeof fetch>>;

beforeEach(() => {
  fetchMock = vi.fn<typeof fetch>().mockResolvedValueOnce(new Response(markdown, { headers: { ETag: loaded } }));
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.replaceChildren();
});

// The note in the editor, which has the focus, with its text changed to ticked; and its Edit button on the page.
async function edited() {
  const notes = new NotesState();
  notes.files = files;
  const shown = new ShownNote(toDo, notes);
  const button = document.createElement('button');
  const field = document.createElement('textarea');
  document.body.append(button, field);
  shown.button(button);
  await shown.load();
  shown.toggle();
  field.focus();
  shown.edit(ticked);
  return { shown, button };
}

// The note in the editor after a save that found it changed since it loaded.
async function stale() {
  const edit = await edited();
  fetchMock.mockResolvedValueOnce(answer(412));
  await edit.shown.save();
  return edit;
}

function answer(status: number): Response {
  return new Response(null, { status });
}

function made(path: string): Response {
  return Response.json({ path }, { status: 201 });
}

// The new note's request, which sends the editor's text.
const post = [url, expect.objectContaining({ method: 'POST', body: ticked })];

test('offers a new note once a save finds the note changed since it loaded, until the editor closes', async () => {
  const { shown } = await edited();
  expect(shown.stale).toBe(false);
  fetchMock.mockResolvedValueOnce(answer(500)).mockResolvedValueOnce(answer(412));

  await shown.save();
  expect(shown.stale).toBe(false);
  await shown.save();
  expect(shown.stale).toBe(true);
  shown.toggle();
  expect(shown.stale).toBe(false);
  shown.toggle();

  expect(shown.stale).toBe(false);
});

test('keeps the text as a new note at the note’s name, then shows the note as it is now and says where', async () => {
  const { shown, button } = await stale();
  fetchMock
    .mockResolvedValueOnce(made('Daily/To do 1.md'))
    .mockResolvedValueOnce(new Response(changed, { headers: { ETag: '"3"' } }));

  const keeping = shown.keep();
  expect(shown.status).toEqual({ kind: 'saving' });
  expect(shown.busy).toBe(true);
  await keeping;

  expect(fetchMock.mock.calls.slice(2)).toEqual([post, [url]]);
  expect(shown.body).toEqual({
    kind: 'loaded',
    markdown: changed,
    version: '"3"',
    note: renderNote(changed, toDo, files),
  });
  expect(shown.editing).toBe(false);
  expect(shown.stale).toBe(false);
  expect(shown.status).toEqual({ kind: 'kept', path: 'Daily/To do 1.md' });
  expect(button).toHaveFocus();
});

test.each([
  { name: 'is missing', reload: () => Promise.resolve(answer(404)) },
  { name: 'fails', reload: () => Promise.resolve(answer(500)) },
  { name: 'can not be reached', reload: () => Promise.reject(new TypeError('Failed to fetch')) },
])('keeps the note it showed when the note $name as it loads again', async ({ reload }) => {
  const { shown } = await stale();
  fetchMock.mockResolvedValueOnce(made('Daily/To do 1.md')).mockImplementationOnce(reload);

  await shown.keep();

  expect(shown.body).toMatchObject({ kind: 'loaded', markdown, version: loaded });
  expect(shown.editing).toBe(false);
  expect(shown.status).toEqual({ kind: 'kept', path: 'Daily/To do 1.md' });
});

test.each([
  { name: 'answers with an error', failure: () => Promise.resolve(answer(409)) },
  { name: 'can not be reached', failure: () => Promise.reject(new TypeError('Failed to fetch')) },
])('keeps the editor, its text and the offer when the gateway $name, and tries again', async ({ failure }) => {
  const { shown } = await stale();
  fetchMock.mockImplementationOnce(failure);

  await shown.keep();
  expect(shown.status).toEqual({ kind: 'failed' });
  expect(shown.editing).toBe(true);
  expect(shown.stale).toBe(true);
  fetchMock.mockResolvedValueOnce(made('Daily/To do 1.md')).mockResolvedValueOnce(new Response(changed));
  await shown.keep();

  expect(fetchMock.mock.calls.slice(2, 4)).toEqual([post, post]);
  expect(shown.status).toEqual({ kind: 'kept', path: 'Daily/To do 1.md' });
});

test('makes no new note before a save finds the note changed', async () => {
  const { shown } = await edited();

  await shown.keep();

  expect(fetchMock).toHaveBeenCalledOnce();
  expect(shown.status).toEqual({ kind: 'idle' });
});

test('makes one new note, and saves nothing, while it runs', async () => {
  const { shown } = await stale();
  const making = Promise.withResolvers<Response>();
  fetchMock.mockReturnValueOnce(making.promise).mockResolvedValueOnce(new Response(changed));
  const keeping = shown.keep();

  await shown.keep();
  await shown.save();
  shown.toggle();
  expect(fetchMock).toHaveBeenCalledTimes(3);
  expect(shown.editing).toBe(true);
  making.resolve(made('Daily/To do 1.md'));
  await keeping;

  expect(fetchMock).toHaveBeenCalledTimes(4);
  expect(shown.status).toEqual({ kind: 'kept', path: 'Daily/To do 1.md' });
});
