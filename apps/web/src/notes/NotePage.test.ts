import { render, screen, within } from '@testing-library/svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import NotePage from './NotePage.svelte';
import { NotesState } from './state.svelte';
import { type Note, noteAt } from './tree';

const toDo = noteAt('/notes/Daily/To%20do')!;
const inbox = noteAt('/notes/Inbox')!;

const markdown = [
  '---',
  'title: To do',
  'tags: [guitar, songs]',
  'album: First Light',
  '---',
  '# To do',
  '',
  '## Today',
  '- [x] Tune',
].join('\n');

// The app's state for the Notes pages, with the vault's list loaded.
function listed(): NotesState {
  const notes = new NotesState();
  notes.files = ['Daily/To do.md', 'Inbox.md'];
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

// The page for the note, with the answer to its request held back until the test gives it.
function held(note: Note) {
  const answer = Promise.withResolvers<Response>();
  fetchMock.mockReturnValueOnce(answer.promise);
  return { answer, page: render(NotePage, { props: { note, notes: listed() } }) };
}

test('shows the head at once, then the properties and the note once it loads', async () => {
  const { answer } = held(toDo);

  expect(screen.getByRole('heading', { level: 1, name: 'To do' })).toBeInTheDocument();
  expect(screen.getByText('Daily')).toBeInTheDocument();
  expect(screen.queryByRole('article')).not.toBeInTheDocument();
  expect(screen.queryByText(/note/)).not.toBeInTheDocument();
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/notes/files/Daily/To%20do.md');

  answer.resolve(new Response(markdown));

  const note = await screen.findByRole('article');
  expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1);
  expect(within(note).getByRole('heading', { level: 2, name: 'Today' })).toBeInTheDocument();
  const box = within(note).getByRole('checkbox');
  expect(box).toBeChecked();
  expect(box).toBeDisabled();
  expect(box.closest('li')).toHaveTextContent('Tune');
});

test('shows the properties: each key, then its text or its chips', async () => {
  render(NotePage, { props: { note: toDo, notes: listed() } });

  const rows = [...(await screen.findByRole('article')).querySelectorAll('dl > div')].map((row) => ({
    key: within(row as HTMLElement).getByRole('term').textContent,
    text: within(row as HTMLElement)
      .getByRole('definition')
      .textContent?.trim(),
    chips: within(row as HTMLElement)
      .queryAllByRole('listitem')
      .map((chip) => chip.textContent),
  }));
  expect(rows).toEqual([
    { key: 'tags', text: 'guitarsongs', chips: ['guitar', 'songs'] },
    { key: 'album', text: 'First Light', chips: [] },
  ]);
});

test('shows no properties block for a note without frontmatter, and no eyebrow at the vault’s top', async () => {
  fetchMock.mockResolvedValueOnce(new Response('Just text'));
  render(NotePage, { props: { note: inbox, notes: listed() } });

  expect(await screen.findByText('Just text')).toBeInTheDocument();
  expect(document.querySelector('dl')).toBeNull();
  expect(document.querySelector('.eyebrow')).toBeNull();
});

test.each([
  {
    name: 'there is no such note',
    answer: () => new Response('no such file', { status: 404 }),
    want: "There's no note at Daily/To do.",
  },
  { name: 'the gateway fails', answer: () => new Response(null, { status: 502 }), want: "Couldn't load the note." },
  {
    name: 'the user is signed out',
    answer: () => new Response(null, { status: 401 }),
    want: "Couldn't load the note.",
  },
])('says so when $name', async ({ answer, want }) => {
  fetchMock.mockResolvedValueOnce(answer());
  render(NotePage, { props: { note: toDo, notes: listed() } });

  expect(await screen.findByText(want)).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 1, name: 'To do' })).toBeInTheDocument();
  expect(screen.queryByRole('article')).not.toBeInTheDocument();
});

test('says it could not load the note when the gateway can not be reached', async () => {
  fetchMock.mockRejectedValueOnce(new TypeError('Failed to fetch'));
  render(NotePage, { props: { note: toDo, notes: listed() } });

  expect(await screen.findByText("Couldn't load the note.")).toBeInTheDocument();
});

test('loads the note again each time its page shows', async () => {
  const first = render(NotePage, { props: { note: toDo, notes: listed() } });
  await screen.findByRole('article');
  first.unmount();

  render(NotePage, { props: { note: toDo, notes: listed() } });
  await screen.findByRole('article');

  expect(fetchMock).toHaveBeenCalledTimes(2);
});

test('shows only the latest note when the page goes straight to another, whichever answer comes last', async () => {
  const { answer: late, page } = held(toDo);
  const next = Promise.withResolvers<Response>();
  fetchMock.mockReturnValueOnce(next.promise);

  await page.rerender({ note: inbox });

  expect(screen.getByRole('heading', { level: 1, name: 'Inbox' })).toBeInTheDocument();
  next.resolve(new Response('Inbox text'));
  expect(await screen.findByText('Inbox text')).toBeInTheDocument();

  late.resolve(new Response('To do text'));
  await new Promise((resolve) => setTimeout(resolve, 0));

  expect(screen.queryByText('To do text')).not.toBeInTheDocument();
  expect(screen.getByText('Inbox text')).toBeInTheDocument();
});

test('clears the last note while the next one loads', async () => {
  const { rerender } = render(NotePage, { props: { note: toDo, notes: listed() } });
  await screen.findByRole('article');
  const next = Promise.withResolvers<Response>();
  fetchMock.mockReturnValueOnce(next.promise);

  await rerender({ note: inbox });

  expect(screen.getByRole('heading', { level: 1, name: 'Inbox' })).toBeInTheDocument();
  expect(screen.queryByRole('article')).not.toBeInTheDocument();
  next.resolve(new Response('Inbox text'));
  expect(await screen.findByText('Inbox text')).toBeInTheDocument();
});

test('links the wikilinks of the note and its properties with the app’s list, and dims those that lead nowhere', async () => {
  fetchMock.mockResolvedValueOnce(
    new Response('---\nnext: "[[Inbox]]"\nlast: "[[Nothing]]"\n---\nSee [[Inbox]] and [[Gone|gone]].')
  );
  render(NotePage, { props: { note: toDo, notes: listed() } });

  const note = await screen.findByRole('article');
  const links = within(note).getAllByRole('link', { name: 'Inbox' });
  expect(links.map((link) => link.getAttribute('href'))).toEqual(['/notes/Inbox', '/notes/Inbox']);
  for (const text of ['Nothing', 'gone']) {
    expect(within(note).getByText(text)).toHaveClass('unresolved');
    expect(within(note).queryByRole('link', { name: text })).not.toBeInTheDocument();
  }
});

// Answers the list's request with what list gives, and the note's with a link to Inbox.
function answering(list: () => Promise<Response>) {
  fetchMock.mockImplementation((input) =>
    input === '/api/notes' ? list() : Promise.resolve(new Response('[[Inbox]]'))
  );
}

test.each([
  { name: 'loads the list', start: () => undefined },
  { name: 'waits for the list’s load that runs', start: (notes: NotesState) => notes.load() },
])('$name when the page finds it not loaded, and asks for it and the note once', async ({ start }) => {
  const list = Promise.withResolvers<Response>();
  answering(() => list.promise);
  const notes = new NotesState();
  const running = start(notes);

  render(NotePage, { props: { note: toDo, notes } });
  list.resolve(Response.json({ files: ['Daily/To do.md', 'Inbox.md'] }));

  expect(await screen.findByRole('link', { name: 'Inbox' })).toHaveAttribute('href', '/notes/Inbox');
  await running;
  expect(fetchMock.mock.calls.map(([input]) => input)).toEqual(['/api/notes', '/api/notes/files/Daily/To%20do.md']);
  expect(notes.status.kind).toBe('loaded');
});

test('says it could not load the note when the list can not load', async () => {
  answering(async () => new Response(null, { status: 502 }));
  render(NotePage, { props: { note: toDo, notes: new NotesState() } });

  expect(await screen.findByText("Couldn't load the note.")).toBeInTheDocument();
});
