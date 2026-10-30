import { afterEach, expect, test, vi } from 'vitest';

import { loadPlans, type Plan, startPlan } from './api';

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

test('starts a plan from the prompt as typed', async () => {
  const fetchMock = answer(Response.json({ id: plans[0]?.id }, { status: 201 }));

  expect(await startPlan(' Plan a trip\n')).toBeUndefined();
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/planner/plans', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{"prompt":" Plan a trip\\n"}',
  });
});

test.each([
  {
    name: 'Copilot off',
    response: new Response('Copilot is not set up\n', { status: 404 }),
    want: "Copilot isn't set up.",
  },
  {
    name: 'a prompt over 64 KiB',
    response: new Response('Request Entity Too Large\n', { status: 413 }),
    want: 'The prompt is too long.',
  },
  {
    name: 'an error from Copilot',
    response: new Response('Bad Gateway\n', { status: 502 }),
    want: "Copilot couldn't start the plan.",
  },
  { name: 'a refusal', response: new Response('the prompt is blank\n', { status: 400 }), want: 'the prompt is blank' },
  { name: 'another answer', response: new Response('', { status: 500 }), want: "The plan didn't start (500)." },
  {
    name: 'a broken connection',
    response: new Error('offline'),
    want: "The plan didn't start: the planner isn't reachable.",
  },
])('says why a start failed on $name', async ({ response, want }) => {
  answer(response);

  expect(await startPlan('Plan a trip')).toBe(want);
});
