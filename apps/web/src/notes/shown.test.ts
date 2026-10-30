import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import { renderNote } from './render';
import { ShownNote } from './shown.svelte';
import { NotesState } from './state.svelte';
import { noteAt } from './tree';

const toDo = noteAt('/notes/Daily/To%20do')!;
const files = ['Daily/To do.md', 'Inbox.md'];
const markdown = '# To do\n\n- [ ] Tune';
const ticked = '# To do\n\n- [x] Tune';
const loaded = '"1"';
const saved = '"2"';

let fetchMock: ReturnType<typeof vi.fn<typeof fetch>>;

beforeEach(() => {
  fetchMock = vi.fn<typeof fetch>().mockResolvedValueOnce(new Response(markdown, { headers: { ETag: loaded } }));
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.replaceChildren();
});

// The note, loaded with the vault's list, and its Edit button on the page.
async function shownNote() {
  const notes = new NotesState();
  notes.files = files;
  const shown = new ShownNote(toDo, notes);
  const button = document.createElement('button');
  document.body.append(button);
  shown.button(button);
  await shown.load();
  return { shown, button };
}

// Presses Edit, and puts the focus in a field, as the editor takes it.
function open(shown: ShownNote): void {
  shown.toggle();
  const field = document.createElement('textarea');
  document.body.append(field);
  field.focus();
}

// The note in the editor, its text changed to ticked.
async function edited() {
  const { shown, button } = await shownNote();
  open(shown);
  shown.edit(ticked);
  return { shown, button };
}

function answer(status: number, version?: string): Response {
  return new Response(null, { status, headers: version === undefined ? {} : { ETag: version } });
}

// The If-Match and the body of the save's request.
function sent(call: number) {
  const init = fetchMock.mock.calls[call]?.[1];
  return { ifMatch: new Headers(init?.headers).get('If-Match'), body: init?.body };
}

test('keeps the note’s version, from its ETag, with its text', async () => {
  const { shown } = await shownNote();

  expect(shown.body).toEqual({ kind: 'loaded', markdown, version: loaded, note: renderNote(markdown, toDo, files) });
});

test('saves the text if the note is still at its version, then shows it, says so and gives Edit the focus', async () => {
  const { shown, button } = await edited();
  fetchMock.mockResolvedValueOnce(answer(204, saved));

  const saving = shown.save();
  expect(shown.status).toBe('saving');
  expect(shown.busy).toBe(true);
  await saving;

  expect(fetchMock).toHaveBeenLastCalledWith(
    '/api/notes/files/Daily/To%20do.md',
    expect.objectContaining({ method: 'PUT' })
  );
  expect(sent(1)).toEqual({ ifMatch: loaded, body: ticked });
  expect(shown.body).toEqual({
    kind: 'loaded',
    markdown: ticked,
    version: saved,
    note: renderNote(ticked, toDo, files),
  });
  expect(shown.editing).toBe(false);
  expect(shown.status).toBe('saved');
  expect(button).toHaveFocus();
});

test('Edit starts again from the saved text, and the next save sends the version the last one gave', async () => {
  const { shown } = await edited();
  fetchMock.mockResolvedValueOnce(answer(204, saved)).mockResolvedValueOnce(answer(204, '"3"'));
  await shown.save();

  shown.toggle();
  expect(shown.status).toBe('idle');
  shown.edit(`${ticked}\n- [ ] Sing`);
  await shown.save();

  expect(sent(2)).toEqual({ ifMatch: saved, body: `${ticked}\n- [ ] Sing` });
});

test.each([
  { name: 'with no change', change: (_: ShownNote) => undefined },
  { name: 'with its changes undone', change: (shown: ShownNote) => shown.edit(markdown) },
])('only closes the editor $name', async ({ change }) => {
  const { shown, button } = await shownNote();
  open(shown);
  change(shown);

  await shown.save();

  expect(fetchMock).toHaveBeenCalledOnce();
  expect(shown.editing).toBe(false);
  expect(shown.status).toBe('idle');
  expect(button).toHaveFocus();
});

test('saves nothing while the editor is closed', async () => {
  const { shown } = await shownNote();

  await shown.save();

  expect(fetchMock).toHaveBeenCalledOnce();
  expect(shown.status).toBe('idle');
});

test('keeps the editor and its text when the note has changed since it loaded, and the next try sends it again', async () => {
  const { shown } = await edited();
  fetchMock.mockResolvedValue(answer(412));

  await shown.save();
  expect(shown.status).toBe('changed');
  expect(shown.editing).toBe(true);
  expect(shown.body).toMatchObject({ markdown, version: loaded });
  await shown.save();

  expect(sent(2)).toEqual({ ifMatch: loaded, body: ticked });
});

test.each([
  { name: 'answers with an error', answer: () => Promise.resolve(answer(500)) },
  { name: 'can not be reached', answer: () => Promise.reject(new TypeError('Failed to fetch')) },
])('keeps the editor and says the save failed when the gateway $name', async ({ answer }) => {
  const { shown } = await edited();
  fetchMock.mockImplementationOnce(answer);

  await shown.save();

  expect(shown.status).toBe('failed');
  expect(shown.editing).toBe(true);
  expect(shown.body).toMatchObject({ markdown, version: loaded });
});

test('Cancel and Save wait while a save runs', async () => {
  const { shown } = await edited();
  const put = Promise.withResolvers<Response>();
  fetchMock.mockReturnValueOnce(put.promise);
  const saving = shown.save();
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));

  shown.toggle();
  await shown.save();
  expect(shown.editing).toBe(true);
  expect(fetchMock).toHaveBeenCalledTimes(2);
  put.resolve(answer(204, saved));
  await saving;

  expect(shown.status).toBe('saved');
});

test('Cancel drops the last save’s word with the edits', async () => {
  const { shown } = await edited();
  fetchMock.mockResolvedValueOnce(answer(500));
  await shown.save();

  shown.toggle();

  expect(shown.editing).toBe(false);
  expect(shown.status).toBe('idle');
  shown.toggle();
  await shown.save();
  expect(fetchMock).toHaveBeenCalledTimes(2);
});
