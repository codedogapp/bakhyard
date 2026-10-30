import { afterEach, expect, test, vi } from 'vitest';

import { PromptState } from './prompt.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

function answer(response: Promise<Response>) {
  const fetchMock = vi.fn<typeof fetch>().mockReturnValue(response);
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const id = '0b6c1f9e-2d4a-4c3b-9a8e-5f7d6c4b3a21';
const started = () => Promise.resolve(Response.json({ id }, { status: 201 }));

test("starts a plan from the prompt, then empties it, says so and passes on the new plan's ID", async () => {
  const fetchMock = answer(started());
  const prompt = new PromptState();
  const onStarted = vi.fn();
  prompt.text = 'Plan a trip';

  await prompt.start(onStarted);

  expect(fetchMock).toHaveBeenCalledOnce();
  expect(prompt.text).toBe('');
  expect(prompt.status).toEqual({ kind: 'started' });
  expect(onStarted).toHaveBeenCalledExactlyOnceWith(id);
});

test('starts nothing for a blank prompt', async () => {
  const fetchMock = answer(started());
  const prompt = new PromptState();
  const onStarted = vi.fn();
  prompt.text = ' \n ';

  await prompt.start(onStarted);

  expect(fetchMock).not.toHaveBeenCalled();
  expect(prompt.status).toEqual({ kind: 'idle' });
  expect(onStarted).not.toHaveBeenCalled();
});

test('starts nothing more while a start runs', async () => {
  const response = Promise.withResolvers<Response>();
  const fetchMock = answer(response.promise);
  const prompt = new PromptState();
  const onStarted = vi.fn();
  prompt.text = 'Plan a trip';

  const first = prompt.start(onStarted);
  expect(prompt.busy).toBe(true);
  await prompt.start(onStarted);
  response.resolve(await started());
  await first;

  expect(fetchMock).toHaveBeenCalledOnce();
  expect(prompt.busy).toBe(false);
  expect(onStarted).toHaveBeenCalledOnce();
});

test('keeps the prompt when the start fails, and says why', async () => {
  answer(Promise.resolve(new Response('Bad Gateway\n', { status: 502 })));
  const prompt = new PromptState();
  const onStarted = vi.fn();
  prompt.text = 'Plan a trip';

  await prompt.start(onStarted);

  expect(prompt.text).toBe('Plan a trip');
  expect(prompt.status).toEqual({ kind: 'failed', reason: "Copilot couldn't start the plan." });
  expect(onStarted).not.toHaveBeenCalled();
});
