import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import NotePage from './NotePage.svelte';
import { NotesState } from './state.svelte';
import { noteAt } from './tree';

const toDo = noteAt('/notes/Daily/To%20do')!;
const markdown = '# To do\n\n- [ ] Tune';
const history = 'https://git.example.com/me/vault/commits/branch/main';

// The app's state for the Notes pages, with the vault's list loaded, from a notes service that saves notes or not and
// has the vault's history or not.
function listed(editable: boolean, history?: string): NotesState {
  const notes = new NotesState();
  notes.files = ['Daily/To do.md', 'Inbox.md'];
  notes.editable = editable;
  notes.history = history;
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

test('links to the note’s history in the same tab, after Edit, once the note loads', async () => {
  const answer = Promise.withResolvers<Response>();
  fetchMock.mockReturnValueOnce(answer.promise);
  render(NotePage, { props: { note: toDo, notes: listed(true, history) } });

  expect(screen.queryByText('History')).not.toBeInTheDocument();
  answer.resolve(new Response(markdown));

  const link = await screen.findByRole('link', { name: 'History' });
  expect(link).toHaveAttribute('href', `${history}/Daily/To%20do.md`);
  expect(link).not.toHaveAttribute('target');
  expect(screen.getByRole('button', { name: 'Edit' }).nextElementSibling).toBe(link);
});

test('links to the note’s history with no Edit when notes saves no note', async () => {
  render(NotePage, { props: { note: toDo, notes: listed(false, history) } });

  expect(await screen.findByRole('link', { name: 'History' })).toHaveAttribute('href', `${history}/Daily/To%20do.md`);
  expect(screen.queryByRole('button')).not.toBeInTheDocument();
});

test('hides the history while the editor is open, since leaving would drop its text', async () => {
  render(NotePage, { props: { note: toDo, notes: listed(true, history) } });
  await fireEvent.click(await screen.findByRole('button', { name: 'Edit' }));

  expect(screen.queryByText('History')).not.toBeInTheDocument();
  await fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
  expect(screen.getByRole('link', { name: 'History' })).toBeInTheDocument();
});

test.each([
  { name: 'notes has none', history: undefined, answer: () => new Response(markdown), want: 'Tune' },
  {
    name: 'there is no such note',
    history,
    answer: () => new Response('no such file', { status: 404 }),
    want: "There's no note at Daily/To do.",
  },
  {
    name: 'the note can not load',
    history,
    answer: () => new Response(null, { status: 502 }),
    want: "Couldn't load the note.",
  },
])('offers no history when $name', async ({ history, answer, want }) => {
  fetchMock.mockResolvedValueOnce(answer());
  render(NotePage, { props: { note: toDo, notes: listed(true, history) } });

  expect(await screen.findByText(want)).toBeInTheDocument();
  expect(screen.queryByText('History')).not.toBeInTheDocument();
});

test('has no row for the tools when notes neither saves notes nor has a history', async () => {
  render(NotePage, { props: { note: toDo, notes: listed(false) } });

  expect(await screen.findByRole('article')).toHaveTextContent('Tune');
  expect(screen.queryByRole('status')).not.toBeInTheDocument();
});
