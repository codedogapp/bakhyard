import { afterEach, expect, test, vi } from 'vitest';

import { loadTickets, type Snapshot } from './tickets';

afterEach(() => {
  vi.unstubAllGlobals();
});

const ticket = {
  key: 'WEB-12',
  summary: 'Show the order history',
  status: 'In Review',
  category: 'in-progress',
  url: 'https://jira.example.com/browse/WEB-12',
} as const;
const shipped = { ...ticket, key: 'WEB-3', status: 'Closed', category: 'done' } as const;

test('loads the tickets from the todos service', async () => {
  const snapshot: Snapshot = { tickets: [ticket], done: [shipped], searched: '2026-10-26T00:41:00Z' };
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(Response.json(snapshot));
  vi.stubGlobal('fetch', fetchMock);

  expect(await loadTickets()).toEqual({ kind: 'loaded', snapshot });
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos/tickets');
});
