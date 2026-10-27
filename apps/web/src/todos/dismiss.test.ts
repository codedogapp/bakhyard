import { afterEach, expect, test, vi } from 'vitest';

import { Focus } from './focus';
import { Inbox } from './inbox.svelte';
import { TodosState } from './state.svelte';
import type { Ticket } from './tickets';

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.replaceChildren();
});

const ticket = (key: string): Ticket => ({
  key,
  summary: `Work on ${key}`,
  status: 'To Do',
  category: 'to-do',
  url: `https://jira.example.com/browse/${key}`,
});
const [review, picker, certificates] = [ticket('WEB-12'), ticket('WEB-9'), ticket('OPS-31')];

/** An inbox over no todos and my open tickets, with a row's Add and Dismiss buttons per ticket, named "add WEB-9". */
function inboxOf(open: Ticket[], dismissed?: string[]) {
  const list = new TodosState();
  list.todos = [];
  const focusAdd = vi.fn<() => void>();
  const inbox = new Inbox(list, () => ({ kind: 'loaded', snapshot: { tickets: open } }), new Focus(focusAdd));
  inbox.dismissed = dismissed && new Set(dismissed);
  const buttons = new Map<string, HTMLButtonElement>();
  for (const { key } of open) {
    for (const action of ['add', 'dismiss'] as const) {
      const button = document.body.appendChild(document.createElement('button'));
      inbox.button(action, key)(button);
      buttons.set(`${action} ${key}`, button);
    }
  }
  return { inbox, buttons, focusAdd };
}

function serve(...answers: (Response | Promise<Response>)[]) {
  const fetchMock = vi.fn<typeof fetch>();
  for (const answer of answers) {
    fetchMock.mockReturnValueOnce(Promise.resolve(answer));
  }
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const dismissed = () => new Response(null, { status: 201 });
const keys = (inbox: Inbox) => inbox.tickets.map(({ key }) => key);

test('loads the dismissed tickets, and leaves them out of the inbox', async () => {
  const fetchMock = serve(Response.json({ dismissals: ['ABC-1', 'WEB-9'] }));
  const { inbox } = inboxOf([review, picker, certificates]);
  expect(inbox.loading).toBe(true);
  expect(inbox.tickets).toEqual([]);

  await inbox.load();

  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos/dismissals');
  expect(inbox.loading).toBe(false);
  expect(inbox.failed).toBe(false);
  expect(keys(inbox)).toEqual(['WEB-12', 'OPS-31']);
});

test.each([
  { failure: 'a 500', answer: () => new Response(null, { status: 500 }) },
  { failure: 'no gateway', answer: () => Promise.reject(new TypeError('Failed to fetch')) },
])('lists no ticket when the dismissed tickets fail to load with $failure', async ({ answer }) => {
  serve(answer());
  const { inbox } = inboxOf([review]);

  await inbox.load();

  expect(inbox.failed).toBe(true);
  expect(inbox.loading).toBe(false);
  expect(inbox.tickets).toEqual([]);
});

test.each([201, 204])('dismisses the ticket, which leaves the inbox, when the service answers %i', async (status) => {
  const fetchMock = serve(new Response(null, { status }));
  const { inbox } = inboxOf([review, picker], ['ABC-1']);

  await inbox.dismiss(review);

  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos/dismissals/WEB-12', { method: 'PUT' });
  expect(keys(inbox)).toEqual(['WEB-9']);
  expect(inbox.dismissed).toEqual(new Set(['ABC-1', 'WEB-12']));
  expect(inbox.statuses.size).toBe(0);
});

test("gives the lost focus to the next row's Dismiss, else the previous row's, else the Add field", async () => {
  serve(dismissed(), dismissed(), dismissed());
  const { inbox, buttons, focusAdd } = inboxOf([review, picker, certificates], []);

  await inbox.dismiss(review);
  expect(document.activeElement).toBe(buttons.get('dismiss WEB-9'));

  buttons.get('dismiss WEB-9')?.blur();
  await inbox.dismiss(certificates);
  expect(document.activeElement).toBe(buttons.get('dismiss WEB-9'));
  expect(focusAdd).not.toHaveBeenCalled();

  buttons.get('dismiss WEB-9')?.blur();
  await inbox.dismiss(picker);
  expect(focusAdd).toHaveBeenCalledOnce();
});

test('dismisses a ticket once at a time, and holds its add, while the dismiss runs', async () => {
  const answer = Promise.withResolvers<Response>();
  const fetchMock = serve(answer.promise);
  const { inbox } = inboxOf([review, picker], []);

  const dismissing = inbox.dismiss(review);
  expect(inbox.busy('WEB-12')).toBe(true);
  expect(inbox.statuses.get('WEB-12')).toEqual({ kind: 'dismissing' });
  expect(inbox.busy('WEB-9')).toBe(false);
  await inbox.dismiss(review);
  await inbox.add(review);
  expect(fetchMock).toHaveBeenCalledOnce();

  answer.resolve(dismissed());
  await dismissing;
  expect(inbox.busy('WEB-12')).toBe(false);
});

test('holds the dismiss while the add runs', async () => {
  const answer = Promise.withResolvers<Response>();
  const fetchMock = serve(answer.promise);
  const { inbox } = inboxOf([review], []);

  const adding = inbox.add(review);
  await inbox.dismiss(review);
  expect(fetchMock).toHaveBeenCalledOnce();
  expect(inbox.statuses.get('WEB-12')).toEqual({ kind: 'adding' });

  answer.resolve(new Response(null, { status: 500 }));
  await adding;
  expect(inbox.statuses.get('WEB-12')).toEqual({ kind: 'failed' });
});

test('keeps a ticket whose dismiss failed in the inbox, and the focus where it was, and dismisses it on the next try', async () => {
  serve(new Response(null, { status: 500 }), Promise.reject(new TypeError('Failed to fetch')), dismissed());
  const { inbox, focusAdd } = inboxOf([review], []);

  await inbox.dismiss(review);
  expect(inbox.statuses.get('WEB-12')).toEqual({ kind: 'dismissFailed' });
  await inbox.dismiss(review);
  expect(inbox.statuses.get('WEB-12')).toEqual({ kind: 'dismissFailed' });
  expect(keys(inbox)).toEqual(['WEB-12']);
  expect(focusAdd).not.toHaveBeenCalled();

  await inbox.dismiss(review);
  expect(inbox.statuses.size).toBe(0);
  expect(inbox.tickets).toEqual([]);
});

test('dismisses several tickets at once, keeping each as its answer lands', async () => {
  const first = Promise.withResolvers<Response>();
  serve(first.promise, dismissed());
  const { inbox } = inboxOf([review, picker, certificates], []);

  const dismissing = inbox.dismiss(review);
  await inbox.dismiss(picker);
  first.resolve(dismissed());
  await dismissing;

  expect(keys(inbox)).toEqual(['OPS-31']);
  expect(inbox.dismissed).toEqual(new Set(['WEB-9', 'WEB-12']));
});
