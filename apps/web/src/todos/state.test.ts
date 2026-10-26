import { afterEach, expect, test, vi } from 'vitest';

import type { Todo } from './api';
import { TodosState } from './state.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

test('has no todos until the list loads', () => {
  const list = new TodosState();

  expect(list.todos).toBeUndefined();
  expect(list.failed).toBe(false);
});

test('loads the todos', async () => {
  const todos = [{ id: 1, title: 'Buy milk', done: false }];
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(Response.json({ todos })));
  const list = new TodosState();

  await list.load();

  expect(list.todos).toEqual(todos);
  expect(list.failed).toBe(false);
});

test('says when the list fails to load', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 500 })));
  const list = new TodosState();

  await list.load();

  expect(list.todos).toBeUndefined();
  expect(list.failed).toBe(true);
});

function loaded(todos: readonly Todo[] | undefined, draft: string): TodosState {
  const list = new TodosState();
  list.todos = todos;
  list.draft = draft;
  return list;
}

const milk = { id: 1, title: 'Buy milk', done: false };
const bank = { id: 2, title: 'Call the bank', done: false };

test('adds the draft on top of the list and empties the draft', async () => {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(Response.json(bank, { status: 201 }));
  vi.stubGlobal('fetch', fetchMock);
  const list = loaded([milk], ' Call the bank ');

  await list.add();

  expect(list.todos).toEqual([bank, milk]);
  expect(list.draft).toBe('');
  expect(list.addStatus).toEqual({ kind: 'added' });
  expect(JSON.parse(String(fetchMock.mock.calls[0][1]?.body))).toEqual({ title: ' Call the bank ' });
});

test('adds one todo at a time', async () => {
  const answer = Promise.withResolvers<Response>();
  const fetchMock = vi.fn<typeof fetch>().mockReturnValue(answer.promise);
  vi.stubGlobal('fetch', fetchMock);
  const list = loaded([], 'Call the bank');

  const first = list.add();
  expect(list.busy).toBe(true);
  expect(list.addStatus).toEqual({ kind: 'adding' });
  await list.add();
  answer.resolve(Response.json(bank, { status: 201 }));
  await first;

  expect(fetchMock).toHaveBeenCalledOnce();
  expect(list.busy).toBe(false);
  expect(list.todos).toEqual([bank]);
});

test.each([
  { name: 'a blank draft', todos: [milk], draft: ' \n ' },
  { name: 'a list that has not loaded', todos: undefined, draft: 'Call the bank' },
])('adds nothing for $name', async ({ todos, draft }) => {
  const fetchMock = vi.fn<typeof fetch>();
  vi.stubGlobal('fetch', fetchMock);
  const list = loaded(todos, draft);

  await list.add();

  expect(fetchMock).not.toHaveBeenCalled();
  expect(list.todos).toEqual(todos);
  expect(list.addStatus).toEqual({ kind: 'idle' });
});

test('keeps the draft and the list when the service refuses the title', async () => {
  const refusal = new Response('invalid title: has a control character\n', { status: 400 });
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(refusal));
  const list = loaded([milk], 'Call\tthe bank');

  await list.add();

  expect(list.addStatus).toEqual({ kind: 'refused', reason: 'invalid title: has a control character' });
  expect(list.draft).toBe('Call\tthe bank');
  expect(list.todos).toEqual([milk]);
});

test('keeps the draft and the list when the add fails', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));
  const list = loaded([milk], 'Call the bank');

  await list.add();

  expect(list.addStatus).toEqual({ kind: 'failed' });
  expect(list.draft).toBe('Call the bank');
  expect(list.todos).toEqual([milk]);
  expect(list.busy).toBe(false);
});

test.each([true, false])('ticks the done mark %s at once, then saves it', async (done) => {
  const answer = Promise.withResolvers<Response>();
  const fetchMock = vi.fn<typeof fetch>().mockReturnValue(answer.promise);
  vi.stubGlobal('fetch', fetchMock);
  const list = loaded([{ ...bank, done: !done }, milk], '');

  const tick = list.tick(2, done);
  expect(list.todos).toEqual([{ ...bank, done }, milk]);
  expect(list.ticking(2)).toBe(true);
  // A rename that the service stored after the tick answers first; the tick's answer has the old title.
  list.retitle({ ...bank, title: 'Call the bank today', done });
  answer.resolve(Response.json({ ...bank, done }));
  await tick;

  expect(list.todos).toEqual([{ ...bank, title: 'Call the bank today', done }, milk]);
  expect(list.ticking(2)).toBe(false);
  expect(list.ticks.size).toBe(0);
  expect(JSON.parse(String(fetchMock.mock.calls[0][1]?.body))).toEqual({ done });
});

test.each([true, false])('puts the done mark back and keeps the failure when ticking it %s fails', async (done) => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));
  const list = loaded([{ ...bank, done: !done }, milk], '');

  await list.tick(2, done);

  expect(list.todos).toEqual([{ ...bank, done: !done }, milk]);
  expect(list.ticks.get(2)).toEqual({ kind: 'failed', done });
  expect(list.ticking(2)).toBe(false);
});

test('ticks a todo once at a time, and the other todos meanwhile', async () => {
  const answer = Promise.withResolvers<Response>();
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockReturnValueOnce(answer.promise)
    .mockResolvedValueOnce(Response.json({ ...milk, done: true }));
  vi.stubGlobal('fetch', fetchMock);
  const list = loaded([bank, milk], '');

  const first = list.tick(2, true);
  await list.tick(2, false);
  await list.tick(1, true);
  answer.resolve(Response.json({ ...bank, done: true }));
  await first;

  expect(fetchMock.mock.calls.map(([url]) => url)).toEqual(['/api/todos/2', '/api/todos/1']);
  expect(list.todos).toEqual([
    { ...bank, done: true },
    { ...milk, done: true },
  ]);
});

test('clears a failed tick with the next one', async () => {
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockRejectedValueOnce(new TypeError('Failed to fetch'))
    .mockResolvedValueOnce(Response.json({ ...bank, done: true }));
  vi.stubGlobal('fetch', fetchMock);
  const list = loaded([bank, milk], '');

  await list.tick(2, true);
  const retry = list.tick(2, true);
  expect(list.ticks.get(2)).toEqual({ kind: 'ticking', done: true });
  await retry;

  expect(list.ticks.has(2)).toBe(false);
  expect(list.todos).toEqual([{ ...bank, done: true }, milk]);
});
