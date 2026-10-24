import { fireEvent, render, screen, within } from '@testing-library/svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import NotesPage from './NotesPage.svelte';
import { NotesState } from './state.svelte';

const files = ['Work/Plans/Q3.md', 'Work/Plans/Day 2.md', 'Work/standup.md', 'Work/diagram.png', 'Inbox.md'];

let fetchMock: ReturnType<typeof vi.fn<typeof fetch>>;

beforeEach(() => {
  fetchMock = vi.fn<typeof fetch>(async () => Response.json({ files }));
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

// The folder whose row has the name.
function folder(name: string): HTMLDetailsElement {
  const details = screen.getByText(name).closest('details');
  expect(details).not.toBeNull();
  return details as HTMLDetailsElement;
}

// The folder's next toggle event, which the browser fires after the click that opens or closes it.
function toggled(name: string): Promise<Event> {
  return new Promise((resolve) => folder(name).addEventListener('toggle', resolve, { once: true }));
}

// Opens or closes the folder as the user does, and waits for its toggle event.
async function toggle(name: string) {
  const event = toggled(name);
  await fireEvent.click(screen.getByText(name));
  await event;
}

test('lists the folders, closed, then the notes once the list loads', async () => {
  const load = Promise.withResolvers<Response>();
  fetchMock.mockReturnValueOnce(load.promise);
  render(NotesPage, { props: { notes: new NotesState() } });

  expect(screen.getByRole('heading', { level: 1, name: 'Notes' })).toBeInTheDocument();
  expect(screen.queryByText(/in the vault\.$/)).not.toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Sync' })).toHaveAttribute('aria-disabled', 'true');

  load.resolve(Response.json({ files }));

  expect(await screen.findByText('4 notes in the vault.')).toBeInTheDocument();
  expect(folder('Work').open).toBe(false);
  expect(within(folder('Work')).getByText('3 notes')).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Inbox' })).toHaveAttribute('href', '/notes/Inbox');
  expect(screen.getByRole('status')).toHaveTextContent('');
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/notes', { method: 'GET' });
});

test('keeps the folders the user opens open for the next time the page shows', async () => {
  const notes = new NotesState();
  const first = render(NotesPage, { props: { notes } });
  await screen.findByText('4 notes in the vault.');

  const opened = Promise.all([toggled('Work'), toggled('Plans')]);

  // Both clicks come before either toggle event, so Plans opens while Work's event is still on its way.
  await fireEvent.click(screen.getByText('Work'));
  await fireEvent.click(screen.getByText('Plans'));
  await opened;

  expect(folder('Work').open).toBe(true);
  expect(folder('Plans').open).toBe(true);
  expect(screen.getByRole('link', { name: 'Day 2' })).toHaveAttribute('href', '/notes/Work/Plans/Day%202');

  first.unmount();
  render(NotesPage, { props: { notes } });

  expect(folder('Work').open).toBe(true);
  expect(folder('Plans').open).toBe(true);
  expect(fetchMock).toHaveBeenCalledOnce();
});

test('keeps a folder the user closes closed for the next time the page shows', async () => {
  const notes = new NotesState();
  const first = render(NotesPage, { props: { notes } });
  await screen.findByText('4 notes in the vault.');

  await toggle('Work');
  await toggle('Work');
  first.unmount();
  render(NotesPage, { props: { notes } });

  expect(folder('Work').open).toBe(false);
});

test('lists the notes that match the filter, each with its folder, in place of the folders', async () => {
  render(NotesPage, { props: { notes: new NotesState() } });
  await screen.findByText('4 notes in the vault.');

  await fireEvent.input(screen.getByRole('searchbox', { name: 'Filter notes' }), { target: { value: 'work d' } });

  const links = screen.getAllByRole('link');
  expect(links.map((link) => [link.textContent, link.getAttribute('href')])).toEqual([
    ['Day 2 Work/Plans', '/notes/Work/Plans/Day%202'],
    ['standup Work', '/notes/Work/standup'],
  ]);
  expect(screen.queryByText('Inbox')).not.toBeInTheDocument();

  await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'inbox' } });

  expect(screen.getByRole('link')).toHaveTextContent(/^Inbox$/);

  await fireEvent.input(screen.getByRole('searchbox'), { target: { value: ' zzz ' } });

  expect(screen.getByText('No notes match “zzz”.')).toBeInTheDocument();

  await fireEvent.input(screen.getByRole('searchbox'), { target: { value: '  ' } });

  expect(folder('Work')).toBeInTheDocument();
});

test('syncs the vault, says when, and lists it as it is now', async () => {
  vi.useFakeTimers({ toFake: ['Date'] });
  // 14:02 in Tokyo, the tests' time zone.
  vi.setSystemTime(new Date('2026-10-24T05:02:00Z'));
  render(NotesPage, { props: { notes: new NotesState() } });
  await screen.findByText('4 notes in the vault.');
  const sync = Promise.withResolvers<Response>();
  fetchMock.mockReturnValueOnce(sync.promise);
  const button = screen.getByRole('button', { name: 'Sync' });

  await fireEvent.click(button);

  expect(screen.getByRole('status')).toHaveTextContent('Syncing…');
  expect(button).toHaveAttribute('aria-disabled', 'true');

  await fireEvent.click(button);

  expect(fetchMock).toHaveBeenCalledTimes(2);

  sync.resolve(Response.json({ files: [...files, 'Later.md'] }));

  expect(await screen.findByText('5 notes in the vault.')).toBeInTheDocument();
  expect(screen.getByRole('status')).toHaveTextContent('Synced at 14:02');
  expect(button).toHaveAttribute('aria-disabled', 'false');
  expect(fetchMock).toHaveBeenLastCalledWith('/api/notes/sync', { method: 'POST' });
  vi.useRealTimers();
});

test('says when a sync fails, and keeps the list', async () => {
  render(NotesPage, { props: { notes: new NotesState() } });
  await screen.findByText('4 notes in the vault.');
  fetchMock.mockResolvedValueOnce(new Response(null, { status: 500 }));

  await fireEvent.click(screen.getByRole('button', { name: 'Sync' }));

  expect(await screen.findByText('Sync failed')).toBeInTheDocument();
  expect(screen.getByText('4 notes in the vault.')).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Inbox' })).toBeInTheDocument();
});

test('says when the list could not load, and lists the vault after a sync', async () => {
  fetchMock.mockResolvedValueOnce(new Response(null, { status: 502 }));
  render(NotesPage, { props: { notes: new NotesState() } });

  expect(await screen.findByText("Couldn't load the notes.")).toBeInTheDocument();
  expect(screen.queryByText(/in the vault\.$/)).not.toBeInTheDocument();

  await fireEvent.click(screen.getByRole('button', { name: 'Sync' }));

  expect(await screen.findByText('4 notes in the vault.')).toBeInTheDocument();
  expect(screen.queryByText("Couldn't load the notes.")).not.toBeInTheDocument();
});

test('shows no list for a vault without notes', async () => {
  fetchMock.mockResolvedValue(Response.json({ files: ['photo.png'] }));
  render(NotesPage, { props: { notes: new NotesState() } });

  expect(await screen.findByText('No notes in the vault.')).toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
});
