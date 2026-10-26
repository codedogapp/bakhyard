import { afterEach, expect, test, vi } from 'vitest';

import { addTodo, fetchTodos } from './api';

afterEach(() => {
  vi.unstubAllGlobals();
});

test('lists the todos through the gateway', async () => {
  const todos = [
    { id: 2, title: 'Call home' },
    { id: 1, title: 'Buy milk' },
  ];
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(Response.json({ todos }));
  vi.stubGlobal('fetch', fetchMock);

  await expect(fetchTodos()).resolves.toEqual(todos);
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos');
});

test.each([401, 500, 502])('fails when the gateway answers with %i', async (status) => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status })));

  await expect(fetchTodos()).rejects.toThrow(`GET /api/todos: ${status}`);
});

test('fails when the gateway can not be reached', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));

  await expect(fetchTodos()).rejects.toThrow('Failed to fetch');
});

test('adds a todo with the title as typed', async () => {
  const todo = { id: 4, title: 'Call the bank' };
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(Response.json(todo, { status: 201 }));
  vi.stubGlobal('fetch', fetchMock);

  await expect(addTodo('  Call the bank ')).resolves.toEqual({ kind: 'added', todo });
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{"title":"  Call the bank "}',
  });
});

test('gives the reason the service refused a title for', async () => {
  const refusal = new Response('invalid title: has a control character\n', { status: 400 });
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(refusal));

  await expect(addTodo('Call\tthe bank')).resolves.toEqual({
    kind: 'refused',
    reason: 'invalid title: has a control character',
  });
});

test.each([200, 401, 413, 500, 502])('fails to add when the gateway answers with %i', async (status) => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(new Response('{}', { status })));

  await expect(addTodo('Call the bank')).rejects.toThrow(`POST /api/todos: ${status}`);
});

test('fails to add when the gateway can not be reached', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));

  await expect(addTodo('Call the bank')).rejects.toThrow('Failed to fetch');
});
