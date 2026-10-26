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
  const todos = [{ id: 1, title: 'Buy milk' }];
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

const milk = { id: 1, title: 'Buy milk' };
const bank = { id: 2, title: 'Call the bank' };

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
