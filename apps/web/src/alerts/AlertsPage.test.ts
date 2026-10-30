import { render, screen, within } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import AlertsPage from './AlertsPage.svelte';
import type { Alert } from './alerts';

afterEach(() => {
  vi.unstubAllGlobals();
});

function answer(response: () => Promise<Response>) {
  const fetchMock = vi.fn<typeof fetch>(response);
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

// Saturday 31 October 2026, 14:00 in Tokyo, where the tests run.
const now = new Date('2026-10-31T14:00:00+09:00');
const alerts: Alert[] = [
  {
    name: 'OnBattery',
    summary: 'The laptop runs on its battery',
    description: 'Plug the charger in',
    startedAt: '2026-10-31T13:05:00+09:00',
  },
  { name: 'PostgresDown', summary: '', description: '', startedAt: '2026-10-29T08:00:00+09:00' },
  {
    name: 'FilesystemFull',
    summary: 'A filesystem is nearly full',
    description: '/ is 91% full',
    startedAt: '2026-10-30T23:30:00+09:00',
    resolvedAt: '2026-10-31T00:10:00+09:00',
  },
];

const section = (head: string) =>
  screen.getByRole('heading', { level: 2, name: head }).closest('section') as HTMLElement;
const rows = (head: string) =>
  within(section(head))
    .getAllByRole('listitem')
    .map((row) => row.textContent?.replaceAll(/\s+/g, ' ').trim());

test('lists the alerts that fire, then those of the last 30 days, as the page shows', async () => {
  const fetchMock = answer(async () => Response.json({ alerts }));
  render(AlertsPage, { props: { now } });

  expect(screen.getByRole('heading', { level: 1, name: 'Alerts' })).toBeInTheDocument();
  expect(await screen.findByRole('heading', { level: 2, name: 'Firing now' })).toBeInTheDocument();
  expect(within(section('Firing now')).getByText('2 alerts')).toBeInTheDocument();
  expect(rows('Firing now')).toEqual([
    'Today The laptop runs on its battery Plug the charger in Since 13:05',
    '29 Oct PostgresDown Since 08:00',
  ]);
  expect(within(section('The last 30 days')).getByText('1 alert')).toBeInTheDocument();
  expect(rows('The last 30 days')).toEqual([
    'Yesterday A filesystem is nearly full / is 91% full 23:30 – 31 Oct, 00:10',
  ]);
  expect(within(section('Firing now')).getByText('Today')).toHaveAttribute('datetime', '2026-10-31T13:05:00+09:00');
  expect(fetchMock).toHaveBeenCalledOnce();
});

test.each([
  ['Firing now', alerts.slice(2), 'No alerts are firing.'],
  ['The last 30 days', alerts.slice(0, 2), 'No alerts in the last 30 days.'],
])('says when the section %s has no alert', async (head, list, empty) => {
  answer(async () => Response.json({ alerts: list }));
  render(AlertsPage, { props: { now } });

  expect(await screen.findByText(empty)).toBeInTheDocument();
  expect(within(section(head)).queryByRole('list')).toBeNull();
});

test("says when the alerts couldn't load, with no section", async () => {
  answer(async () => new Response('Bad Gateway\n', { status: 502 }));
  render(AlertsPage, { props: { now } });

  expect(await screen.findByText("Couldn't load the alerts.")).toBeInTheDocument();
  expect(screen.queryByRole('heading', { level: 2 })).toBeNull();
});

test('shows only its head while the alerts load', () => {
  answer(() => new Promise(() => {}));
  render(AlertsPage, { props: { now } });

  expect(screen.getByRole('heading', { level: 1, name: 'Alerts' })).toBeInTheDocument();
  expect(screen.queryByRole('heading', { level: 2 })).toBeNull();
  expect(screen.queryByText("Couldn't load the alerts.")).toBeNull();
});
