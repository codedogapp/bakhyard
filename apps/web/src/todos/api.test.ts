import { afterEach, expect, test, vi } from 'vitest';

import { fetchTodos } from './api';

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
