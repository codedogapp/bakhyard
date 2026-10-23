import { afterEach, expect, test, vi } from 'vitest';

import { fetchHome, SignedOutError } from './api';

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
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 503 })));

  const failure = fetchHome();

  await expect(failure).rejects.toThrow('GET /api/home: 503');
  await expect(failure).rejects.not.toBeInstanceOf(SignedOutError);
});

test('fails as signed out when the user has no session', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 401 })));

  const failure = fetchHome();

  await expect(failure).rejects.toBeInstanceOf(SignedOutError);
  await expect(failure).rejects.toThrow('GET /api/home: 401');
});
