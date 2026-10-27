import { fireEvent, render, screen, within } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import type { Todo } from './api';
import { Focus } from './focus';
import { Inbox } from './inbox.svelte';
import JiraInbox from './JiraInbox.svelte';
import { TodosState } from './state.svelte';
import type { Ticket } from './tickets';

afterEach(() => {
  vi.unstubAllGlobals();
});

const review: Ticket = {
  key: 'WEB-12',
  summary: 'Show the <b>order</b> history',
  status: 'In Review',
  category: 'in-progress',
  url: 'https://jira.example.com/browse/WEB-12',
};
const certificates: Ticket = {
  key: 'OPS-31',
  summary: 'Rotate the staging certificates',
  status: 'To Do',
  category: 'to-do',
  url: 'https://jira.example.com/browse/OPS-31',
};

/** Shows the inbox of my open tickets over the todos, none dismissed, giving the focus to the Add field with `focusAdd`. */
function show(todos: Todo[], open: Ticket[], focusAdd = vi.fn<() => void>()) {
  const list = new TodosState();
  list.todos = todos;
  const inbox = new Inbox(list, () => ({ kind: 'loaded', snapshot: { tickets: open } }), new Focus(focusAdd));
  inbox.dismissed = new Set();
  render(JiraInbox, { props: { inbox } });
  return list;
}

const rows = () => within(screen.getByRole('list')).getAllByRole('listitem');
const text = (element: HTMLElement) => element.textContent?.replaceAll(/\s+/g, ' ').trim();
const addButton = (key: string) => screen.getByRole('button', { name: `Add ${key}` });
const dismissButton = (key: string) => screen.getByRole('button', { name: `Dismiss ${key}` });
const statusOf = (row: number) => screen.getAllByRole('status')[row];

test('lists the tickets that no todo names, in their order, under a head that counts them', () => {
  const linked: Todo = { id: 1, title: 'Fix the date picker', done: true, priority: 'normal', ticket: 'WEB-9' };
  show([linked], [review, { ...certificates, key: 'WEB-9' }, certificates]);

  expect(screen.getByRole('heading', { level: 2, name: 'Jira inbox' })).toBeInTheDocument();
  expect(screen.getByText('2 tickets')).toBeInTheDocument();
  expect(rows().map(text)).toEqual([
    'WEB-12 Show the <b>order</b> history In Review Add Dismiss',
    'OPS-31 Rotate the staging certificates To Do Add Dismiss',
  ]);
  expect(rows()[0]?.querySelector('.chip')).toHaveClass('chip-filled');
  expect(screen.getAllByRole('status').map((status) => status.textContent)).toEqual(['', '']);
  expect(screen.getAllByRole('button').map((button) => button.getAttribute('aria-label'))).toEqual([
    'Add WEB-12',
    'Dismiss WEB-12',
    'Add OPS-31',
    'Dismiss OPS-31',
  ]);
  for (const button of screen.getAllByRole('button')) {
    expect(button).toHaveAttribute('aria-disabled', 'false');
  }
});

test('shows no section when every open ticket has its todo', () => {
  show([{ id: 1, title: 'Rotate', done: false, priority: 'low', ticket: 'OPS-31' }], [certificates]);

  expect(screen.queryByRole('heading')).toBeNull();
  expect(screen.queryByRole('list')).toBeNull();
});

test('adds the ticket as a todo, which takes its row off, and the section with its last row', async () => {
  const todo: Todo = { id: 7, title: certificates.summary, done: false, priority: 'normal', ticket: 'OPS-31' };
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(Response.json(todo, { status: 201 })));
  const list = show([], [review, certificates]);

  await fireEvent.click(addButton('OPS-31'));
  await vi.waitFor(() =>
    expect(rows().map(text)).toEqual(['WEB-12 Show the <b>order</b> history In Review Add Dismiss'])
  );
  expect(screen.getByText('1 ticket')).toBeInTheDocument();
  expect(list.todos).toEqual([todo]);

  vi.stubGlobal(
    'fetch',
    vi.fn<typeof fetch>().mockResolvedValue(Response.json({ ...todo, id: 8, ticket: 'WEB-12' }, { status: 201 }))
  );
  await fireEvent.click(addButton('WEB-12'));
  await vi.waitFor(() => expect(screen.queryByRole('heading')).toBeNull());
});

test("says the row's add runs, and why it was refused or failed, while the row stays", async () => {
  const answer = Promise.withResolvers<Response>();
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockReturnValueOnce(answer.promise));
  show([], [review, certificates]);

  await fireEvent.click(addButton('WEB-12'));
  expect(statusOf(0)).toHaveTextContent('Adding…');
  expect(addButton('WEB-12')).toHaveAttribute('aria-disabled', 'true');
  expect(dismissButton('WEB-12')).toHaveAttribute('aria-disabled', 'true');
  expect(addButton('OPS-31')).toHaveAttribute('aria-disabled', 'false');

  answer.resolve(new Response('invalid title: has a control character\n', { status: 400 }));
  await vi.waitFor(() => expect(statusOf(0)).toHaveTextContent('invalid title: has a control character'));
  expect(addButton('WEB-12')).toHaveAttribute('aria-disabled', 'false');

  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 500 })));
  await fireEvent.click(addButton('OPS-31'));
  await vi.waitFor(() => expect(statusOf(1)).toHaveTextContent('Add failed'));
  expect(rows()).toHaveLength(2);
});

test("gives the lost focus to a row that's still there when adds cross", async () => {
  const first = Promise.withResolvers<Response>();
  const todo: Todo = { id: 7, title: certificates.summary, done: false, priority: 'normal', ticket: 'OPS-31' };
  const fetchMock = vi.fn<typeof fetch>().mockReturnValueOnce(first.promise);
  vi.stubGlobal('fetch', fetchMock.mockResolvedValueOnce(Response.json(todo, { status: 201 })));
  const focusAdd = vi.fn<() => void>();
  show([], [review, certificates], focusAdd);

  addButton('WEB-12').focus();
  await fireEvent.click(addButton('WEB-12'));
  addButton('OPS-31').focus();
  await fireEvent.click(addButton('OPS-31'));
  await vi.waitFor(() => expect(addButton('WEB-12')).toHaveFocus());

  first.resolve(Response.json({ ...todo, id: 8, ticket: 'WEB-12' }, { status: 201 }));
  await vi.waitFor(() => expect(screen.queryByRole('list')).toBeNull());
  expect(focusAdd).toHaveBeenCalledOnce();
});

test('dismisses the ticket, which takes its row off, and the section with its last row', async () => {
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockImplementation(() => Promise.resolve(new Response(null, { status: 201 })));
  vi.stubGlobal('fetch', fetchMock);
  show([], [review, certificates]);

  await fireEvent.click(dismissButton('OPS-31'));
  await vi.waitFor(() =>
    expect(rows().map(text)).toEqual(['WEB-12 Show the <b>order</b> history In Review Add Dismiss'])
  );
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos/dismissals/OPS-31', { method: 'PUT' });
  expect(screen.getByText('1 ticket')).toBeInTheDocument();

  await fireEvent.click(dismissButton('WEB-12'));
  await vi.waitFor(() => expect(screen.queryByRole('heading')).toBeNull());
});

test("says the row's dismiss runs, and that it failed, while both its buttons wait and the row stays", async () => {
  const answer = Promise.withResolvers<Response>();
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockReturnValueOnce(answer.promise));
  show([], [review, certificates]);

  await fireEvent.click(dismissButton('WEB-12'));
  expect(statusOf(0)).toHaveTextContent('Dismissing…');
  expect(addButton('WEB-12')).toHaveAttribute('aria-disabled', 'true');
  expect(dismissButton('WEB-12')).toHaveAttribute('aria-disabled', 'true');
  expect(dismissButton('OPS-31')).toHaveAttribute('aria-disabled', 'false');

  answer.resolve(new Response(null, { status: 500 }));
  await vi.waitFor(() => expect(statusOf(0)).toHaveTextContent('Dismiss failed'));
  expect(addButton('WEB-12')).toHaveAttribute('aria-disabled', 'false');
  expect(dismissButton('WEB-12')).toHaveAttribute('aria-disabled', 'false');
  expect(rows()).toHaveLength(2);
});

test("gives the lost focus to the next row's Dismiss after a dismiss", async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 201 })));
  show([], [review, certificates]);

  dismissButton('WEB-12').focus();
  await fireEvent.click(dismissButton('WEB-12'));
  await vi.waitFor(() => expect(dismissButton('OPS-31')).toHaveFocus());
  expect(rows()).toHaveLength(1);
});
