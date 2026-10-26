import { fireEvent, render, screen, within } from '@testing-library/svelte';
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
  expect(screen.queryByRole('textbox')).not.toBeInTheDocument();

  const todos = [
    { id: 3, title: 'Call <b>home</b>' },
    { id: 1, title: 'Buy milk' },
  ];
  load.resolve(Response.json({ todos }));

  expect(await screen.findByText('2 todos on the list.')).toBeInTheDocument();
  const rows = within(screen.getByRole('list')).getAllByRole('listitem');
  expect(rows.map((row) => row.textContent)).toEqual(['Call <b>home</b>', 'Buy milk']);
  expect(screen.getByRole('textbox', { name: 'New todo' })).toBeInTheDocument();
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos');
});

test('shows no list without todos, until the first is added', async () => {
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockResolvedValueOnce(Response.json({ todos: [] }))
    .mockResolvedValueOnce(Response.json({ id: 1, title: 'Buy milk' }, { status: 201 }));
  vi.stubGlobal('fetch', fetchMock);
  render(TodosPage);

  expect(await screen.findByText('No todos on the list.')).toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();

  await fireEvent.input(screen.getByRole('textbox', { name: 'New todo' }), { target: { value: 'Buy milk' } });
  await fireEvent.click(screen.getByRole('button', { name: 'Add' }));

  expect(await screen.findByText('1 todo on the list.')).toBeInTheDocument();
  expect(within(screen.getByRole('list')).getByRole('listitem')).toHaveTextContent('Buy milk');
});

test('says when the list could not load', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));
  render(TodosPage);

  expect(await screen.findByText("Couldn't load the todos.")).toBeInTheDocument();
  expect(screen.queryByText(/on the list\.$/)).not.toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
  expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
});
