import { afterEach, expect, test, vi } from 'vitest';

import { loadSnapshot, type Polls, searchedLabel } from './snapshot';

afterEach(() => {
  vi.unstubAllGlobals();
});

interface Snapshot extends Polls {
  things?: string[];
}

function answer(response: Response) {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(response);
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

test.each<[string, Snapshot]>([
  ['what a poll found', { things: ['a thing'], searched: '2026-10-26T00:41:00Z' }],
  ['nothing found', { things: [], searched: '2026-10-26T00:41:00Z' }],
  ['what a poll found and a failure', { things: ['a thing'], searched: '2026-10-26T00:41:00Z', failure: '502' }],
  ['a failure before any poll succeeded', { failure: 'search: 401 Unauthorized' }],
])('loads a snapshot with %s', async (_, snapshot) => {
  const fetchMock = answer(Response.json(snapshot));

  expect(await loadSnapshot<Snapshot>('/api/things')).toEqual({ kind: 'loaded', snapshot });
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/things');
});

test('finds the source off when the service answers 404', async () => {
  answer(new Response('Things are not set up\n', { status: 404 }));

  expect(await loadSnapshot('/api/things')).toEqual({ kind: 'off' });
});

test.each<[string, Response]>([
  ['an error', new Response('Bad Gateway\n', { status: 502 })],
  ['an error with a snapshot', Response.json({ searched: '2026-10-26T00:41:00Z' }, { status: 401 })],
  ['another success than a 200', Response.json({ searched: '2026-10-26T00:41:00Z' }, { status: 201 })],
  ['a body that is not JSON', new Response('<html>')],
  ['null', Response.json(null)],
  ['a snapshot with neither when a poll began nor a failure', Response.json({})],
  ['what a poll found without when it began', Response.json({ things: [] })],
])('fails on %s', async (_, response) => {
  answer(response);

  expect(await loadSnapshot('/api/things')).toEqual({ kind: 'failed' });
});

test('fails when the gateway cannot be reached', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));

  expect(await loadSnapshot('/api/things')).toEqual({ kind: 'failed' });
});

test('tells when a search began in the time zone of the device, Tokyo in the tests', () => {
  expect(searchedLabel('2026-10-26T00:41:00Z')).toBe('26 October at 09:41');
  expect(searchedLabel('2026-12-31T15:05:00Z')).toBe('1 January at 00:05');
});
