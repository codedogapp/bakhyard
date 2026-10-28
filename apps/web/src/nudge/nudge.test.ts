import { afterEach, expect, test, vi } from 'vitest';

import { loadState, sendAnswer } from './nudge';

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

const state = { questions: [], sessions: [] };

test('loads the state', async () => {
  const fetchMock = answer(Response.json(state));

  expect(await loadState()).toEqual({ kind: 'loaded', state });
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/nudge/state');
});

test.each([
  { name: 'a status', response: new Response('', { status: 502 }) },
  { name: 'a broken connection', response: new Error('offline') },
])('fails to load on $name', async ({ response }) => {
  answer(response);

  expect(await loadState()).toEqual({ kind: 'failed' });
});

test("sends an answer to the question's route", async () => {
  const fetchMock = answer(new Response(null, { status: 204 }));

  expect(await sendAnswer('a/b', { option: 'Yes' })).toBeUndefined();
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/nudge/questions/a%2Fb/answer', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{"option":"Yes"}',
  });
});

test.each([
  { response: new Response('', { status: 404 }), want: 'The agent no longer waits for this question.' },
  { response: new Response('unknown option\n', { status: 400 }), want: 'unknown option' },
  { response: new Response('', { status: 500 }), want: 'The answer failed (500).' },
  { response: new Error('offline'), want: "The answer didn't go through: nudge isn't reachable." },
])('says why an answer failed: $want', async ({ response, want }) => {
  answer(response);

  expect(await sendAnswer('1', { text: 'x' })).toBe(want);
});
