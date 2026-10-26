import { render, screen, within } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import TodosPage from './TodosPage.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

test('lists the todos, newest first, once the list loads', async () => {
  const load = Promise.withResolvers<Response>();
  const fetchMock = vi.fn<typeof fetch>().mockReturnValue(load.promise);
  vi.stubGlobal('fetch', fetchMock);
  render(TodosPage);

  expect(screen.getByRole('heading', { level: 1, name: 'Todos' })).toBeInTheDocument();
  expect(screen.queryByText(/on the list\.$/)).not.toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();

  const todos = [
    { id: 3, title: 'Call <b>home</b>' },
    { id: 1, title: 'Buy milk' },
  ];
  load.resolve(Response.json({ todos }));

  expect(await screen.findByText('2 todos on the list.')).toBeInTheDocument();
  const rows = within(screen.getByRole('list')).getAllByRole('listitem');
  expect(rows.map((row) => row.textContent)).toEqual(['Call <b>home</b>', 'Buy milk']);
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos');
});

test('shows no list without todos', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(Response.json({ todos: [] })));
  render(TodosPage);

  expect(await screen.findByText('No todos on the list.')).toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
});

test('says when the list could not load', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));
  render(TodosPage);

  expect(await screen.findByText("Couldn't load the todos.")).toBeInTheDocument();
  expect(screen.queryByText(/on the list\.$/)).not.toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
});
