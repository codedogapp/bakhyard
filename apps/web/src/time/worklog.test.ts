import { afterEach, expect, test, vi } from 'vitest';

import { addWorklog } from './worklog';

afterEach(() => {
  vi.unstubAllGlobals();
});

test('adds a worklog of the minutes on the ticket on the date', async () => {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 201 }));
  vi.stubGlobal('fetch', fetchMock);

  await expect(addWorklog('WEB-9', '2026-10-16', 480)).resolves.toEqual({ kind: 'added' });
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/time/worklogs', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{"ticket":"WEB-9","date":"2026-10-16","minutes":480}',
  });
});

test('gives the reason the service refused the worklog for', async () => {
  const refusal = new Response('invalid date: not one of the last 7 days\n', { status: 400 });
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(refusal));

  await expect(addWorklog('WEB-9', '2026-10-01', 480)).resolves.toEqual({
    kind: 'refused',
    reason: 'invalid date: not one of the last 7 days',
  });
});

test.each([200, 204, 401, 404, 500, 502])('fails when the gateway answers with %i', async (status) => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status })));

  await expect(addWorklog('WEB-9', '2026-10-16', 480)).rejects.toThrow(`POST /api/time/worklogs: ${status}`);
});

test('fails when the gateway can not be reached', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));

  await expect(addWorklog('WEB-9', '2026-10-16', 480)).rejects.toThrow('Failed to fetch');
});
