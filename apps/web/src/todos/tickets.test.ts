import { afterEach, expect, test, vi } from 'vitest';

import type { Todo } from './api';
import { inboxTickets, loadTickets, openTickets, type Snapshot, type Tickets } from './tickets';

afterEach(() => {
  vi.unstubAllGlobals();
});

function answer(response: Response) {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(response);
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const ticket = {
  key: 'WEB-12',
  summary: 'Show the order history',
  status: 'In Review',
  category: 'in-progress',
  url: 'https://jira.example.com/browse/WEB-12',
} as const;

test.each<[string, Snapshot]>([
  ['the tickets', { tickets: [ticket], searched: '2026-10-26T00:41:00Z' }],
  ['no tickets', { tickets: [], searched: '2026-10-26T00:41:00Z' }],
  ['the tickets and a failure', { tickets: [ticket], searched: '2026-10-26T00:41:00Z', failure: 'search Jira: 502' }],
  ['a failure before any search succeeded', { failure: 'search Jira: 401 Unauthorized' }],
])('loads a snapshot with %s', async (_, snapshot) => {
  const fetchMock = answer(Response.json(snapshot));

  expect(await loadTickets()).toEqual({ kind: 'loaded', snapshot });
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos/tickets');
});

test('finds Jira off when the service answers 404', async () => {
  answer(new Response('Jira is not set up\n', { status: 404 }));

  expect(await loadTickets()).toEqual({ kind: 'off' });
});

test.each<[string, Response]>([
  ['an error', new Response('Bad Gateway\n', { status: 502 })],
  ['an error with a snapshot', Response.json({ tickets: [] }, { status: 401 })],
  ['another success than a 200', Response.json({ tickets: [] }, { status: 201 })],
  ['a body that is not JSON', new Response('<html>')],
  ['null', Response.json(null)],
  ['a snapshot with neither tickets nor a failure', Response.json({})],
])('fails on %s', async (_, response) => {
  answer(response);

  expect(await loadTickets()).toEqual({ kind: 'failed' });
});

test('fails when the gateway cannot be reached', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));

  expect(await loadTickets()).toEqual({ kind: 'failed' });
});

const certificates = { ...ticket, key: 'OPS-31', status: 'To Do', category: 'to-do' } as const;

test.each<[string, Tickets, string[]]>([
  ['loaded tickets', { kind: 'loaded', snapshot: { tickets: [ticket, certificates] } }, ['WEB-12', 'OPS-31']],
  [
    'the tickets of a search that succeeded before the last one failed',
    { kind: 'loaded', snapshot: { tickets: [ticket], failure: 'search Jira: 502' } },
    ['WEB-12'],
  ],
  ['no open tickets', { kind: 'loaded', snapshot: { tickets: [] } }, []],
  ['a failure before any search succeeded', { kind: 'loaded', snapshot: { failure: 'search Jira: 401' } }, []],
  ['an empty snapshot, as a stopping service answers', { kind: 'loaded', snapshot: {} }, []],
  ['tickets still loading', { kind: 'loading' }, []],
  ['Jira off', { kind: 'off' }, []],
  ['tickets that failed to load', { kind: 'failed' }, []],
])('finds my open tickets by key in %s', (_, tickets, keys) => {
  const open = openTickets(tickets);

  expect([...open.keys()]).toEqual(keys);
  for (const key of keys) {
    expect(open.get(key)?.key).toBe(key);
  }
});

const picker = { ...ticket, key: 'WEB-9', status: 'In Progress' } as const;
const loaded: Tickets = { kind: 'loaded', snapshot: { tickets: [ticket, picker, certificates] } };
const todo = (ticket?: string): Todo => ({ id: 1, title: 'Buy milk', done: false, priority: 'normal', ticket });

test.each<[string, Tickets, Todo[] | undefined, string[]]>([
  ['no todo names any', loaded, [todo(), todo('ABC-1')], ['WEB-12', 'WEB-9', 'OPS-31']],
  ['todos name some', loaded, [todo('WEB-9'), todo(), todo('WEB-12')], ['OPS-31']],
  ['a done todo names one', loaded, [{ ...todo('WEB-12'), done: true }], ['WEB-9', 'OPS-31']],
  ['todos name them all', loaded, [todo('OPS-31'), todo('WEB-9'), todo('WEB-12')], []],
  ['the todos are still loading', loaded, undefined, []],
  ['a failure before any search succeeded', { kind: 'loaded', snapshot: { failure: 'search Jira: 401' } }, [], []],
  ['tickets still loading', { kind: 'loading' }, [], []],
  ['Jira off', { kind: 'off' }, [], []],
  ['tickets that failed to load', { kind: 'failed' }, [], []],
])('finds the inbox in my open tickets when %s', (_, tickets, todos, keys) => {
  expect(inboxTickets(tickets, todos).map((each) => each.key)).toEqual(keys);
});
