import { afterEach, expect, test, vi } from 'vitest';

import { loadPlans, type Plan, savePlan, sendAnswer, sendMessage, startPlan } from './api';

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

test("starts a plan from the prompt as typed, and returns the new plan's ID", async () => {
  const fetchMock = answer(Response.json({ id: plans[0]?.id }, { status: 201 }));

  expect(await startPlan(' Plan a trip\n')).toEqual({ kind: 'started', id: plans[0]?.id });
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
  {
    name: 'a start whose answer breaks off',
    response: new Response('{"id":', { status: 201 }),
    want: "The plan didn't start: the planner isn't reachable.",
  },
])('says why a start failed on $name', async ({ response, want }) => {
  answer(response);

  expect(await startPlan('Plan a trip')).toEqual({ kind: 'failed', reason: want });
});

test("saves the plan's plan in the vault, and returns the new note's path", async () => {
  const fetchMock = answer(Response.json({ path: 'Plans/2026-10-30-trip.md' }, { status: 201 }));

  expect(await savePlan('p1')).toEqual({ kind: 'saved', path: 'Plans/2026-10-30-trip.md' });
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/planner/plans/p1/note', { method: 'POST' });
});

test.each([
  {
    name: 'no plan yet',
    response: new Response("Copilot hasn't written the plan yet\n", { status: 409 }),
    want: "Copilot hasn't written the plan yet.",
  },
  {
    name: 'an error from notes',
    response: new Response('Bad Gateway\n', { status: 502 }),
    want: "Couldn't save the plan in the vault.",
  },
  {
    name: 'no saving',
    response: new Response('Saving to the vault is not set up\n', { status: 404 }),
    want: 'Saving to the vault is not set up',
  },
  { name: 'no such plan', response: new Response('no such plan\n', { status: 404 }), want: 'no such plan' },
  { name: 'another answer', response: new Response('', { status: 500 }), want: "The plan wasn't saved (500)." },
  {
    name: 'a broken connection',
    response: new Error('offline'),
    want: "The plan wasn't saved: the planner isn't reachable.",
  },
])('says why a save failed, for $name', async ({ response, want }) => {
  answer(response);

  expect(await savePlan('p1')).toEqual({ kind: 'failed', reason: want });
});

test('sends the text as typed to the plan as its next message', async () => {
  const fetchMock = answer(new Response(null, { status: 204 }));

  expect(await sendMessage('a%2Fb', ' Make it a week\n')).toBeUndefined();
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/planner/plans/a%2Fb/messages', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{"text":" Make it a week\\n"}',
  });
});

test.each([
  {
    name: 'a plan that Copilot no longer keeps',
    response: new Response('no such plan\n', { status: 404 }),
    want: 'Copilot no longer has this plan.',
  },
  {
    name: 'a message over 64 KiB',
    response: new Response('Request Entity Too Large\n', { status: 413 }),
    want: 'The message is too long.',
  },
  {
    name: 'an error from Copilot',
    response: new Response('Bad Gateway\n', { status: 502 }),
    want: "Copilot couldn't take the message.",
  },
  {
    name: 'a refusal',
    response: new Response('the message is blank\n', { status: 400 }),
    want: 'the message is blank',
  },
  { name: 'another answer', response: new Response('', { status: 500 }), want: "The message wasn't sent (500)." },
  {
    name: 'a broken connection',
    response: new Error('offline'),
    want: "The message wasn't sent: the planner isn't reachable.",
  },
])('says why a send failed on $name', async ({ response, want }) => {
  answer(response);

  expect(await sendMessage('p1', 'Make it a week')).toBe(want);
});

test('sends the text as typed as the answer to the question, whose ID it escapes', async () => {
  const fetchMock = answer(new Response(null, { status: 204 }));

  expect(await sendAnswer('a%2Fb', 'q/1', ' Rome\n')).toBeUndefined();
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/planner/plans/a%2Fb/questions/q%2F1/answer', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{"text":" Rome\\n"}',
  });
});

test.each([
  {
    name: 'a question that no longer waits',
    response: new Response('no such question\n', { status: 404 }),
    want: 'Copilot no longer waits for this question.',
  },
  {
    name: 'an answer over 64 KiB',
    response: new Response('Request Entity Too Large\n', { status: 413 }),
    want: 'The answer is too long.',
  },
  {
    name: 'an error from Copilot',
    response: new Response('Bad Gateway\n', { status: 502 }),
    want: "Copilot couldn't take the answer.",
  },
  {
    name: 'a refusal',
    response: new Response('the answer is not one of the choices\n', { status: 400 }),
    want: 'the answer is not one of the choices',
  },
  { name: 'another answer', response: new Response('', { status: 500 }), want: "The answer wasn't sent (500)." },
  {
    name: 'a broken connection',
    response: new Error('offline'),
    want: "The answer wasn't sent: the planner isn't reachable.",
  },
])('says why an answer failed on $name', async ({ response, want }) => {
  answer(response);

  expect(await sendAnswer('p1', 'q1', 'Rome')).toBe(want);
});
