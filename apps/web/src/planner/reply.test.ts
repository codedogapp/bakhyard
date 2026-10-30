import { afterEach, expect, test, vi } from 'vitest';

import { ReplyState } from './reply.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

function answer(response: Promise<Response>) {
  const fetchMock = vi.fn<typeof fetch>().mockReturnValue(response);
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const sent = () => Promise.resolve(new Response(null, { status: 204 }));

test('sends the reply to the plan, then empties it and says so', async () => {
  const fetchMock = answer(sent());
  const reply = new ReplyState();
  reply.text = 'Make it a week';

  await reply.send('p1');

  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/planner/plans/p1/messages', expect.anything());
  expect(reply.text).toBe('');
  expect(reply.status).toEqual({ kind: 'sent' });
});

test('sends nothing for a blank reply', async () => {
  const fetchMock = answer(sent());
  const reply = new ReplyState();
  reply.text = ' \n ';

  await reply.send('p1');

  expect(fetchMock).not.toHaveBeenCalled();
  expect(reply.status).toEqual({ kind: 'idle' });
});

test('sends nothing more while a send runs', async () => {
  const response = Promise.withResolvers<Response>();
  const fetchMock = answer(response.promise);
  const reply = new ReplyState();
  reply.text = 'Make it a week';

  const first = reply.send('p1');
  expect(reply.busy).toBe(true);
  await reply.send('p1');
  response.resolve(await sent());
  await first;

  expect(fetchMock).toHaveBeenCalledOnce();
  expect(reply.busy).toBe(false);
  expect(reply.status).toEqual({ kind: 'sent' });
});

test('keeps the reply when the send fails, and says why', async () => {
  answer(Promise.resolve(new Response('Bad Gateway\n', { status: 502 })));
  const reply = new ReplyState();
  reply.text = 'Make it a week';

  await reply.send('p1');

  expect(reply.text).toBe('Make it a week');
  expect(reply.status).toEqual({ kind: 'failed', reason: "Copilot couldn't take the message." });
});
