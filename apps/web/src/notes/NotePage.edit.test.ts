import { EditorView } from '@codemirror/view';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import NotePage from './NotePage.svelte';
import { NotesState } from './state.svelte';
import { noteAt } from './tree';

const toDo = noteAt('/notes/Daily/To%20do')!;
const inbox = noteAt('/notes/Inbox')!;
const markdown = '# To do\n\n- [ ] Tune';

// The app's state for the Notes pages, with the vault's list loaded, from a notes service that saves notes or not.
function listed(editable: boolean): NotesState {
  const notes = new NotesState();
  notes.files = ['Daily/To do.md', 'Inbox.md'];
  notes.editable = editable;
  return notes;
}

let fetchMock: ReturnType<typeof vi.fn<typeof fetch>>;

beforeEach(() => {
  fetchMock = vi.fn<typeof fetch>(async () => new Response(markdown));
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

// The editor's field, once its chunk has loaded.
function field(): Promise<HTMLElement> {
  return screen.findByRole('textbox', { name: 'Markdown' });
}

// Opens the note's page from a notes service that saves notes, and presses Edit once the note loads.
async function editing() {
  const page = render(NotePage, { props: { note: toDo, notes: listed(true) } });
  await fireEvent.click(await screen.findByRole('button', { name: 'Edit' }));
  return { page, view: EditorView.findFromDOM(await field())! };
}

test.each([
  { name: 'the notes service saves no note', editable: false, answer: () => new Response(markdown), want: 'Tune' },
  {
    name: 'there is no such note',
    editable: true,
    answer: () => new Response('no such file', { status: 404 }),
    want: "There's no note at Daily/To do.",
  },
  {
    name: 'the note can not load',
    editable: true,
    answer: () => new Response(null, { status: 502 }),
    want: "Couldn't load the note.",
  },
])('offers no Edit when $name', async ({ editable, answer, want }) => {
  fetchMock.mockResolvedValueOnce(answer());
  render(NotePage, { props: { note: toDo, notes: listed(editable) } });

  expect(await screen.findByText(want)).toBeInTheDocument();
  expect(screen.queryByRole('button')).not.toBeInTheDocument();
});

test('offers Edit once the note loads, which shows its Markdown in the editor in place of the note', async () => {
  const answer = Promise.withResolvers<Response>();
  fetchMock.mockReturnValueOnce(answer.promise);
  render(NotePage, { props: { note: toDo, notes: listed(true) } });

  expect(screen.queryByRole('button')).not.toBeInTheDocument();
  answer.resolve(new Response(markdown));
  await fireEvent.click(await screen.findByRole('button', { name: 'Edit' }));

  const editor = await field();
  expect(editor).toHaveFocus();
  expect([...editor.querySelectorAll('.cm-line')].map((line) => line.textContent)).toEqual([
    '# To do',
    '',
    '- [ ] Tune',
  ]);
  expect(screen.queryByRole('article')).not.toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 1, name: 'To do' })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
});

test('Cancel drops the edits and shows the note again, and Edit starts over from the note', async () => {
  const { view } = await editing();
  view.dispatch({ changes: { from: 0, insert: 'Typed. ' } });
  const button = screen.getByRole('button', { name: 'Cancel' });

  await fireEvent.click(button);

  expect(await screen.findByRole('article')).toHaveTextContent('Tune');
  expect(screen.queryByRole('textbox', { name: 'Markdown' })).not.toBeInTheDocument();
  expect(button).toHaveTextContent('Edit');
  expect(button).toHaveFocus();
  await fireEvent.click(button);
  expect(EditorView.findFromDOM(await field())?.state.doc.toString()).toBe(markdown);
  expect(fetchMock).toHaveBeenCalledOnce();
});

test('leaves the editor when the page goes to another note', async () => {
  const { page } = await editing();
  fetchMock.mockResolvedValueOnce(new Response('Inbox text'));

  await page.rerender({ note: inbox });

  expect(await screen.findByText('Inbox text')).toBeInTheDocument();
  expect(screen.queryByRole('textbox', { name: 'Markdown' })).not.toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Edit' })).toBeInTheDocument();
});

test('says so when the editor can not load, and Cancel shows the note again', async () => {
  vi.doMock('./NoteEditor.svelte', () => {
    throw new Error('Failed to fetch dynamically imported module');
  });
  try {
    render(NotePage, { props: { note: toDo, notes: listed(true) } });
    await fireEvent.click(await screen.findByRole('button', { name: 'Edit' }));

    expect(await screen.findByText("Couldn't load the editor.")).toBeInTheDocument();
    await fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(await screen.findByRole('article')).toHaveTextContent('Tune');
  } finally {
    vi.doUnmock('./NoteEditor.svelte');
  }
});
