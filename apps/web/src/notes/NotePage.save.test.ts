import { flushSync } from 'svelte';
import { EditorView } from '@codemirror/view';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import NotePage from './NotePage.svelte';
import { NotesState } from './state.svelte';
import { noteAt } from './tree';

const toDo = noteAt('/notes/Daily/To%20do')!;
const markdown = '# To do\n\n- [ ] Tune';
const typedText = `${markdown} now`;

let fetchMock: ReturnType<typeof vi.fn<typeof fetch>>;

beforeEach(() => {
  fetchMock = vi.fn<typeof fetch>().mockResolvedValueOnce(new Response(markdown, { headers: { ETag: '"1"' } }));
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

// The note's page from a notes service that saves notes, with the note in the editor and " now" typed at its end.
async function typed() {
  const notes = new NotesState();
  notes.files = ['Daily/To do.md'];
  notes.editable = true;
  render(NotePage, { props: { note: toDo, notes } });
  await fireEvent.click(await screen.findByRole('button', { name: 'Edit' }));
  const view = EditorView.findFromDOM(await screen.findByRole('textbox', { name: 'Markdown' }))!;
  view.dispatch({ changes: { from: markdown.length, insert: ' now' }, userEvent: 'input.type' });
  return { notes, view };
}

function button(name: string): HTMLElement {
  return screen.getByRole('button', { name });
}

test('Save follows Cancel, and saves the typed text, then shows the note, says so and gives Edit the focus', async () => {
  await typed();
  expect(screen.getAllByRole('button').map((each) => each.textContent?.trim())).toEqual(['Cancel', 'Save']);
  expect(screen.getByRole('status').textContent).toBe('');
  fetchMock.mockResolvedValueOnce(new Response(null, { status: 204, headers: { ETag: '"2"' } }));

  await fireEvent.click(button('Save'));

  expect(await screen.findByRole('article')).toHaveTextContent('Tune now');
  expect(fetchMock).toHaveBeenLastCalledWith(
    '/api/notes/files/Daily/To%20do.md',
    expect.objectContaining({ method: 'PUT', body: typedText })
  );
  expect(screen.getByRole('status')).toHaveTextContent('Saved');
  expect(button('Edit')).toHaveFocus();
  expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();
});

test('says so while a save runs, when Save, Cancel and the editor wait, and when the note has changed', async () => {
  const { view } = await typed();
  const put = Promise.withResolvers<Response>();
  fetchMock.mockReturnValueOnce(put.promise);

  await fireEvent.click(button('Save'));

  expect(screen.getByRole('status')).toHaveTextContent('Saving…');
  expect(button('Save')).toHaveAttribute('aria-disabled', 'true');
  expect(button('Cancel')).toHaveAttribute('aria-disabled', 'true');
  view.dispatch({ changes: { from: 0, insert: 'Late. ' }, userEvent: 'input.type' });
  expect(view.state.doc.toString()).toBe(typedText);
  put.resolve(new Response(null, { status: 412 }));
  expect(await screen.findByText('Not saved: the note has changed since it loaded')).toBeInTheDocument();
  expect(button('Save')).toHaveAttribute('aria-disabled', 'false');
  expect(view.state.doc.toString()).toBe(typedText);
});

test('offers to save the text as a new note once the note has changed, then shows the note and says where', async () => {
  await typed();
  const post = Promise.withResolvers<Response>();
  fetchMock
    .mockResolvedValueOnce(new Response(null, { status: 412 }))
    .mockReturnValueOnce(post.promise)
    .mockResolvedValueOnce(new Response(`${markdown}\n- [ ] Sing`, { headers: { ETag: '"3"' } }));
  await fireEvent.click(button('Save'));
  const offer = await screen.findByRole('button', { name: 'Save as new note' });
  expect(screen.getAllByRole('button').map((each) => each.textContent?.trim())).toEqual([
    'Cancel',
    'Save',
    'Save as new note',
  ]);

  await fireEvent.click(offer);
  expect(screen.getByRole('status')).toHaveTextContent('Saving…');
  expect(offer).toHaveAttribute('aria-disabled', 'true');
  post.resolve(Response.json({ path: 'Daily/To do 1.md' }, { status: 201 }));

  expect(await screen.findByRole('article')).toHaveTextContent('Sing');
  expect(fetchMock).toHaveBeenNthCalledWith(
    3,
    '/api/notes/files/Daily/To%20do.md',
    expect.objectContaining({ method: 'POST', body: typedText })
  );
  expect(screen.getByRole('status')).toHaveTextContent('Saved as “To do 1”');
  expect(button('Edit')).toHaveFocus();
  expect(screen.queryByRole('button', { name: 'Save as new note' })).not.toBeInTheDocument();
});

test('keeps Save and Cancel while the editor is open, when a sync says that notes no longer saves', async () => {
  const { notes } = await typed();

  flushSync(() => {
    notes.editable = false;
  });
  expect(button('Save')).toBeInTheDocument();
  await fireEvent.click(button('Cancel'));

  expect(await screen.findByRole('article')).toHaveTextContent('Tune');
  expect(screen.queryByRole('button')).not.toBeInTheDocument();
});
