import { afterEach, expect, test, vi } from 'vitest';

import { fetchHome } from './api';

afterEach(() => {
  vi.unstubAllGlobals();
});

test('fetches the Home through the gateway', async () => {
  const home = { name: 'Ada', items: [{ late: true }] };
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(Response.json(home));
  vi.stubGlobal('fetch', fetchMock);

  await expect(fetchHome()).resolves.toEqual(home);
  expect(fetchMock).toHaveBeenCalledWith('/api/home');
});

test('fails when the gateway answers with an error', async () => {
  const reload = vi.fn();
  vi.stubGlobal('location', { reload });
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 503 })));

  await expect(fetchHome()).rejects.toThrow('GET /api/home: 503');
  expect(reload).not.toHaveBeenCalled();
});

test('reloads the page to sign in again when the session has ended', async () => {
  const reload = vi.fn();
  vi.stubGlobal('location', { reload });
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 401 })));

  await expect(fetchHome()).rejects.toThrow('GET /api/home: 401');
  expect(reload).toHaveBeenCalledOnce();
});
