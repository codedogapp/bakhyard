import { afterEach, expect, test, vi } from 'vitest';

import type { Priority, Todo } from './api';
import { TodosState } from './state.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

function todo(id: number, priority: Priority): Todo {
  return { id, title: `Todo ${id}`, done: false, priority };
}

// High 6 and 4; Normal 5 and 3; Low 1.
const list = [todo(6, 'high'), todo(5, 'normal'), todo(4, 'high'), todo(3, 'normal'), todo(1, 'low')];

function loaded(todos: Todo[] = list): TodosState {
  const state = new TodosState();
  state.todos = todos;
  return state;
}

function serve(answer: Response | Promise<Response>) {
  const fetchMock = vi.fn<typeof fetch>().mockReturnValueOnce(Promise.resolve(answer));
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const ok = () => Response.json({});
const rows = (state: TodosState) => state.todos?.map(({ id, priority }) => `${id} ${priority}`);

test.each([
  {
    id: 6,
    target: { priority: 'high', place: { after: 4 } },
    body: '{"after":4}',
    want: ['5 normal', '4 high', '6 high', '3 normal', '1 low'],
  },
  {
    id: 6,
    target: { priority: 'normal', place: { after: 3 } },
    body: '{"after":3,"priority":"normal"}',
    want: ['5 normal', '4 high', '3 normal', '6 normal', '1 low'],
  },
  {
    id: 1,
    target: { priority: 'high', place: { before: 6 } },
    body: '{"before":6,"priority":"high"}',
    want: ['1 high', '6 high', '5 normal', '4 high', '3 normal'],
  },
] as const)('drops todo $id at $target.place with $body', async ({ id, target, body, want }) => {
  const fetchMock = serve(ok());
  const state = loaded();

  await state.drop(id, target);

  expect(fetchMock).toHaveBeenCalledExactlyOnceWith(
    `/api/todos/${id}`,
    expect.objectContaining({ method: 'PATCH', body })
  );
  expect(rows(state)).toEqual(want);
  expect(state.dropFailed).toBeUndefined();
});

test('moves the todo before the service answers, and is dropping until it does', async () => {
  const answer = Promise.withResolvers<Response>();
  serve(answer.promise);
  const state = loaded();

  const drop = state.drop(1, { priority: 'normal', place: { before: 5 } });

  expect(rows(state)).toEqual(['6 high', '1 normal', '5 normal', '4 high', '3 normal']);
  expect(state.dropping).toBe(true);
  answer.resolve(ok());
  await drop;
  expect(state.dropping).toBe(false);
});

test.each([
  { name: 'refuses it', fetchMock: vi.fn<typeof fetch>().mockResolvedValue(new Response('', { status: 500 })) },
  { name: 'is out of reach', fetchMock: vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')) },
])('puts the todo back and says so when the service $name', async ({ fetchMock }) => {
  vi.stubGlobal('fetch', fetchMock);
  const state = loaded();

  await state.drop(4, { priority: 'low', place: { after: 1 } });

  expect(rows(state)).toEqual(['6 high', '5 normal', '4 high', '3 normal', '1 low']);
  expect(state.dropFailed).toBe(4);
  expect(state.dropping).toBe(false);
});

test.each([
  { id: 1, target: { priority: 'high', place: { before: 6 } } },
  { id: 6, target: { priority: 'low', place: { after: 1 } } },
] as const)('puts todo $id back after a failure, at the end of the list it was at', async ({ id, target }) => {
  serve(new Response('', { status: 404 }));
  const state = loaded();

  await state.drop(id, target);

  expect(state.todos).toEqual(list);
});

test('puts back the priority of a todo that had no neighbour', async () => {
  serve(new Response('', { status: 500 }));
  const state = loaded([todo(1, 'low')]);

  await state.drop(1, { priority: 'high', place: { before: 2 } });

  expect(rows(state)).toEqual(['1 low']);
});

test('forgets the failure at the next drop', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValueOnce(new Response('', { status: 500 })));
  const state = loaded();
  await state.drop(4, { priority: 'high', place: { before: 6 } });
  expect(state.dropFailed).toBe(4);
  serve(ok());

  await state.drop(3, { priority: 'normal', place: { before: 5 } });

  expect(state.dropFailed).toBeUndefined();
});

test('drops nothing for a todo it lacks', async () => {
  const fetchMock = serve(ok());
  const state = loaded();

  await state.drop(7, { priority: 'high', place: { before: 6 } });

  expect(fetchMock).not.toHaveBeenCalled();
  expect(state.todos).toBe(list);
  expect(state.dropping).toBe(false);
});

test('drops nothing before the list loads', async () => {
  const fetchMock = serve(ok());
  const state = new TodosState();

  await state.drop(6, { priority: 'high', place: { before: 4 } });

  expect(fetchMock).not.toHaveBeenCalled();
  expect(state.todos).toBeUndefined();
});
