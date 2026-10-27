import { afterEach, expect, test, vi } from 'vitest';

import { dismissTicket, fetchDismissals } from './api';

afterEach(() => {
  vi.unstubAllGlobals();
});

test('lists the dismissed tickets through the gateway', async () => {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(Response.json({ dismissals: ['OPS-31', 'WEB-9'] }));
  vi.stubGlobal('fetch', fetchMock);

  await expect(fetchDismissals()).resolves.toEqual(['OPS-31', 'WEB-9']);
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos/dismissals');
});

test.each([401, 404, 500, 502])(
  'fails to list the dismissed tickets when the gateway answers with %i',
  async (status) => {
    vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status })));

    await expect(fetchDismissals()).rejects.toThrow(`GET /api/todos/dismissals: ${status}`);
  }
);

test('fails to list the dismissed tickets when the answer is not JSON', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response('<html>')));

  await expect(fetchDismissals()).rejects.toThrow(SyntaxError);
});

test('fails to list the dismissed tickets when the gateway can not be reached', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));

  await expect(fetchDismissals()).rejects.toThrow('Failed to fetch');
});

test.each([201, 204])('dismisses a ticket through the gateway, which answers %i', async (status) => {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status }));
  vi.stubGlobal('fetch', fetchMock);

  await expect(dismissTicket('WEB-9')).resolves.toBeUndefined();
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos/dismissals/WEB-9', { method: 'PUT' });
});

test('escapes the key in the path', async () => {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 201 }));
  vi.stubGlobal('fetch', fetchMock);

  await dismissTicket('A/B?C#D');
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos/dismissals/A%2FB%3FC%23D', { method: 'PUT' });
});

test.each([200, 400, 401, 404, 405, 500, 502])('fails to dismiss when the gateway answers with %i', async (status) => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status })));

  await expect(dismissTicket('WEB-9')).rejects.toThrow(`PUT /api/todos/dismissals/WEB-9: ${status}`);
});

test('fails to dismiss when the gateway can not be reached', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));

  await expect(dismissTicket('WEB-9')).rejects.toThrow('Failed to fetch');
});
