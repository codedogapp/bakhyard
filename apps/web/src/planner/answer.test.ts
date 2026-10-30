import { afterEach, expect, test, vi } from 'vitest';

import { AnswerState } from './answer.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

function answer(response: Promise<Response>) {
  const fetchMock = vi.fn<typeof fetch>().mockReturnValue(response);
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const sent = () => Promise.resolve(new Response(null, { status: 204 }));
const body = (fetchMock: ReturnType<typeof answer>) => fetchMock.mock.calls[0]?.[1]?.body;

test('sends the choice as it is as the answer to the question, and says so', async () => {
  const fetchMock = answer(sent());
  const state = new AnswerState();
  state.text = 'Milan';

  await state.send('p1', 'q1', 'Rome');

  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/planner/plans/p1/questions/q1/answer', expect.anything());
  expect(body(fetchMock)).toBe('{"text":"Rome"}');
  expect(state.status).toEqual({ kind: 'sent' });
});

test('sends the text as typed without a choice, and keeps it till the question goes', async () => {
  const fetchMock = answer(sent());
  const state = new AnswerState();
  state.text = ' Milan,\nthen Rome ';

  await state.send('p1', 'q1');

  expect(body(fetchMock)).toBe('{"text":" Milan,\\nthen Rome "}');
  expect(state.text).toBe(' Milan,\nthen Rome ');
  expect(state.status).toEqual({ kind: 'sent' });
});

test.each([
  { about: 'a blank text', text: ' \n ', choice: undefined },
  { about: 'a blank choice', text: 'Milan', choice: ' ' },
])('sends nothing for $about', async ({ text, choice }) => {
  const fetchMock = answer(sent());
  const state = new AnswerState();
  state.text = text;

  await state.send('p1', 'q1', choice);

  expect(fetchMock).not.toHaveBeenCalled();
  expect(state.status).toEqual({ kind: 'idle' });
});

test('sends nothing more while a send runs', async () => {
  const response = Promise.withResolvers<Response>();
  const fetchMock = answer(response.promise);
  const state = new AnswerState();

  const first = state.send('p1', 'q1', 'Rome');
  expect(state.busy).toBe(true);
  expect(state.status).toEqual({ kind: 'sending' });
  await state.send('p1', 'q1', 'Paris');
  response.resolve(await sent());
  await first;

  expect(fetchMock).toHaveBeenCalledOnce();
  expect(body(fetchMock)).toBe('{"text":"Rome"}');
  expect(state.busy).toBe(false);
});

test('keeps the text when the send fails, and says why', async () => {
  answer(Promise.resolve(new Response('no such question\n', { status: 404 })));
  const state = new AnswerState();
  state.text = 'Milan';

  await state.send('p1', 'q1');

  expect(state.text).toBe('Milan');
  expect(state.status).toEqual({ kind: 'failed', reason: 'Copilot no longer waits for this question.' });
});
