import { afterEach, expect, test, vi } from 'vitest';

import type { Todo } from './api';
import { Focus } from './focus';
import { Inbox } from './inbox.svelte';
import { TodosState } from './state.svelte';
import type { Ticket } from './tickets';

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.replaceChildren();
});

const review: Ticket = {
  key: 'WEB-12',
  summary: 'Show the order history',
  status: 'In Review',
  category: 'in-progress',
  url: 'https://jira.example.com/browse/WEB-12',
};
const picker: Ticket = { ...review, key: 'WEB-9', summary: 'Fix the date picker', status: 'In Progress' };
const certificates: Ticket = { ...review, key: 'OPS-31', summary: 'Rotate the certificates', category: 'to-do' };
const milk: Todo = { id: 1, title: 'Buy milk', done: false, priority: 'normal' };

/** The todo that the service adds for the ticket. */
const todoOf = (ticket: Ticket, id = 7): Todo => ({
  id,
  title: ticket.summary,
  done: false,
  priority: 'normal',
  ticket: ticket.key,
});

/** An inbox over a list of the todos, my open tickets and none dismissed, with a row's Add button per ticket. */
function inboxOf(todos: Todo[] | undefined, open: Ticket[]) {
  const list = new TodosState();
  list.todos = todos;
  const focusAdd = vi.fn<() => void>();
  const inbox = new Inbox(list, () => ({ kind: 'loaded', snapshot: { tickets: open } }), new Focus(focusAdd));
  inbox.dismissed = new Set();
  const buttons = new Map(open.map(({ key }) => [key, document.body.appendChild(document.createElement('button'))]));
  for (const [key, button] of buttons) {
    inbox.button('add', key)(button);
  }
  return { list, inbox, buttons, focusAdd };
}

function serve(...answers: (Response | Promise<Response>)[]) {
  const fetchMock = vi.fn<typeof fetch>();
  for (const answer of answers) {
    fetchMock.mockReturnValueOnce(Promise.resolve(answer));
  }
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const added = (todo: Todo) => Response.json(todo, { status: 201 });
const keys = (inbox: Inbox) => inbox.tickets.map((ticket) => ticket.key);

test('lists my open tickets that no todo names, as the todos change', () => {
  const linked = { ...milk, ticket: 'WEB-9' };
  const { list, inbox } = inboxOf([linked], [review, picker, certificates]);
  expect(keys(inbox)).toEqual(['WEB-12', 'OPS-31']);

  list.remove(linked.id);
  expect(keys(inbox)).toEqual(['WEB-12', 'WEB-9', 'OPS-31']);
  list.prepend(todoOf(certificates));
  expect(keys(inbox)).toEqual(['WEB-12', 'WEB-9']);
  list.applyEdit({ ...todoOf(certificates), ticket: undefined });
  expect(keys(inbox)).toEqual(['WEB-12', 'WEB-9', 'OPS-31']);
});

test('lists no ticket until the todos load', () => {
  expect(inboxOf(undefined, [review]).inbox.tickets).toEqual([]);
});

test('adds the ticket as a todo titled with its summary, on top of the list, which takes it off the inbox', async () => {
  const fetchMock = serve(added(todoOf(review)));
  const { list, inbox } = inboxOf([milk], [review, picker]);

  await inbox.add(review);

  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{"title":"Show the order history","ticket":"WEB-12"}',
  });
  expect(list.todos).toEqual([todoOf(review), milk]);
  expect(keys(inbox)).toEqual(['WEB-9']);
  expect(inbox.statuses.size).toBe(0);
});

test("gives the lost focus to the next row's Add, else the previous row's, else the Add field", async () => {
  serve(added(todoOf(review, 7)), added(todoOf(certificates, 8)), added(todoOf(picker, 9)));
  const { inbox, buttons, focusAdd } = inboxOf([], [review, picker, certificates]);

  await inbox.add(review);
  expect(document.activeElement).toBe(buttons.get('WEB-9'));

  buttons.get('WEB-9')?.blur();
  await inbox.add(certificates);
  expect(document.activeElement).toBe(buttons.get('WEB-9'));
  expect(focusAdd).not.toHaveBeenCalled();

  buttons.get('WEB-9')?.blur();
  await inbox.add(picker);
  expect(focusAdd).toHaveBeenCalledOnce();
});

test('leaves the focus where the user moved it while the add ran', async () => {
  serve(added(todoOf(review)));
  const { inbox, buttons, focusAdd } = inboxOf([], [review, picker]);
  const elsewhere = document.body.appendChild(document.createElement('input'));
  elsewhere.focus();

  await inbox.add(review);

  expect(document.activeElement).toBe(elsewhere);
  expect(buttons.get('WEB-9')).not.toBe(document.activeElement);
  expect(focusAdd).not.toHaveBeenCalled();
});

test('adds a ticket once at a time, and says so while it runs', async () => {
  const answer = Promise.withResolvers<Response>();
  const fetchMock = serve(answer.promise);
  const { inbox } = inboxOf([], [review, picker]);

  const adding = inbox.add(review);
  expect(inbox.busy('WEB-12')).toBe(true);
  expect(inbox.statuses.get('WEB-12')).toEqual({ kind: 'adding' });
  expect(inbox.busy('WEB-9')).toBe(false);
  await inbox.add(review);
  expect(fetchMock).toHaveBeenCalledOnce();

  answer.resolve(added(todoOf(review)));
  await adding;
  expect(inbox.busy('WEB-12')).toBe(false);
});

test("keeps a refused ticket in the inbox, with the service's reason, and the focus where it was", async () => {
  serve(new Response('invalid title: has a control character\n', { status: 400 }));
  const { list, inbox, focusAdd } = inboxOf([milk], [review]);

  await inbox.add(review);

  expect(inbox.statuses.get('WEB-12')).toEqual({ kind: 'refused', reason: 'invalid title: has a control character' });
  expect(keys(inbox)).toEqual(['WEB-12']);
  expect(list.todos).toEqual([milk]);
  expect(focusAdd).not.toHaveBeenCalled();
});

test('keeps a ticket whose add failed in the inbox, and adds it on the next try', async () => {
  serve(new Response(null, { status: 500 }), Promise.reject(new TypeError('Failed to fetch')), added(todoOf(review)));
  const { list, inbox } = inboxOf([milk], [review]);

  await inbox.add(review);
  expect(inbox.statuses.get('WEB-12')).toEqual({ kind: 'failed' });
  await inbox.add(review);
  expect(inbox.statuses.get('WEB-12')).toEqual({ kind: 'failed' });
  expect(keys(inbox)).toEqual(['WEB-12']);

  await inbox.add(review);
  expect(inbox.statuses.size).toBe(0);
  expect(list.todos).toEqual([todoOf(review), milk]);
});

test('adds several tickets at once, each on top as its answer lands', async () => {
  const first = Promise.withResolvers<Response>();
  serve(first.promise, added(todoOf(picker, 8)));
  const { list, inbox } = inboxOf([milk], [review, picker]);

  const adding = inbox.add(review);
  await inbox.add(picker);
  first.resolve(added(todoOf(review, 7)));
  await adding;

  expect(list.todos).toEqual([todoOf(review, 7), todoOf(picker, 8), milk]);
  expect(inbox.tickets).toEqual([]);
});
