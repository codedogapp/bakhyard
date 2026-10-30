import { afterEach, expect, test, vi } from 'vitest';

import { loadPlans, type Plan } from './api';

afterEach(() => {
  vi.unstubAllGlobals();
});

function answer(response: Response | Error) {
  const fetchMock = vi.fn<typeof fetch>(async () => {
    if (response instanceof Error) {
      throw response;
    }
    return response;
  });
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const plans: Plan[] = [
  {
    id: '0b6c1f9e-2d4a-4c3b-9a8e-5f7d6c4b3a21',
    title: 'Back up the home lab',
    started_at: '2026-10-30T23:50:00Z',
    updated_at: '2026-10-31T00:20:00.123456789Z',
  },
];

test('loads the plans from the planner', async () => {
  const fetchMock = answer(Response.json({ plans }));

  expect(await loadPlans()).toEqual({ kind: 'loaded', plans });
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/planner/plans');
});

test.each([
  { name: "the planner's", body: 'Copilot is not set up\n' },
  { name: "the gateway's, which doesn't forward to the planner", body: '404 page not found\n' },
])('is off on a 404, $name', async ({ body }) => {
  answer(new Response(body, { status: 404 }));

  expect(await loadPlans()).toEqual({ kind: 'off' });
});

test.each([
  { name: 'an error from Copilot', response: new Response('Bad Gateway\n', { status: 502 }) },
  { name: 'a body that is not JSON', response: new Response('<html>', { status: 200 }) },
  { name: 'a broken connection', response: new Error('offline') },
])('fails to load on $name', async ({ response }) => {
  answer(response);

  expect(await loadPlans()).toEqual({ kind: 'failed' });
});
