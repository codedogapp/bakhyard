import { afterEach, expect, test, vi } from 'vitest';

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
