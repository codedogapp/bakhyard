import { afterEach, expect, test, vi } from 'vitest';

import { type Alert, alertTitle, loadAlerts, spanLabel } from './alerts';

afterEach(() => {
  vi.unstubAllGlobals();
});

function answer(response: () => Promise<Response>) {
  const fetchMock = vi.fn<typeof fetch>(response);
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

// The tests run in Tokyo, at UTC+9.
const onBattery: Alert = {
  name: 'OnBattery',
  summary: 'The laptop runs on its battery',
  description: 'Plug the charger in',
  startedAt: '2026-10-31T04:05:00.123456Z',
};
const postgresDown: Alert = { name: 'PostgresDown', summary: '', description: '', startedAt: '2026-10-29T23:00:00Z' };
const filesystemFull: Alert = {
  name: 'FilesystemFull',
  summary: 'A filesystem is nearly full',
  description: '/ is 91% full',
  startedAt: '2026-10-30T14:30:00Z',
  resolvedAt: '2026-10-30T15:10:00Z',
};
const cpuHot: Alert = {
  name: 'CPUHot',
  summary: 'The CPU runs hot',
  description: 'Look at what keeps it busy',
  startedAt: '2026-10-19T01:00:00+09:00',
  resolvedAt: '2026-10-19T10:00:00+09:00',
};

test('loads the alerts, those that fire apart from those that resolved, each in the order they come', async () => {
  const fetchMock = answer(async () => Response.json({ alerts: [onBattery, postgresDown, filesystemFull, cpuHot] }));

  expect(await loadAlerts()).toEqual({
    kind: 'loaded',
    firing: [onBattery, postgresDown],
    resolved: [filesystemFull, cpuHot],
  });
  expect(fetchMock).toHaveBeenCalledWith('/api/notify/alerts');
});

test('loads no alerts when none has fired lately', async () => {
  answer(async () => Response.json({ alerts: [] }));

  expect(await loadAlerts()).toEqual({ kind: 'loaded', firing: [], resolved: [] });
});

test.each([
  ['an error', async () => new Response('Internal Server Error\n', { status: 500 })],
  ['an error with a list in its body', async () => Response.json({ alerts: [] }, { status: 502 })],
  ['no gateway route', async () => new Response('404 page not found\n', { status: 404 })],
  ['a body that is not JSON', async () => new Response('<!doctype html>', { status: 200 })],
  [
    'no answer',
    async () => {
      throw new TypeError('Failed to fetch');
    },
  ],
])('a load that gets %s fails', async (_, response) => {
  answer(response);

  expect(await loadAlerts()).toEqual({ kind: 'failed' });
});

test("calls an alert by its summary, or by its rule's name when it has none", () => {
  expect(alertTitle(onBattery)).toBe('The laptop runs on its battery');
  expect(alertTitle(postgresDown)).toBe('PostgresDown');
});

test.each([
  ['an alert that fires', onBattery, 'Since 13:05'],
  ['an alert that resolved the day it fired', cpuHot, '01:00–10:00'],
  ['an alert that resolved the next day', filesystemFull, '23:30 – 31 Oct, 00:10'],
])('says when %s fired and resolved, in the time zone of the device', (_, alert, label) => {
  expect(spanLabel(alert)).toBe(label);
});
