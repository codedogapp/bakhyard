import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
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
    { id: 3, title: 'Call <b>home</b>', done: false },
    { id: 1, title: 'Buy milk', done: true },
  ];
  load.resolve(Response.json({ todos }));

  expect(await screen.findByText('2 todos on the list.')).toBeInTheDocument();
  const rows = within(screen.getByRole('list')).getAllByRole('listitem');
  expect(rows.map((row) => row.textContent?.trim())).toEqual(['Call <b>home</b>', 'Buy milk']);
  expect(screen.getByRole('textbox', { name: 'New todo' })).toBeInTheDocument();
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos');
});

test('shows no list without todos, until the first is added', async () => {
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockResolvedValueOnce(Response.json({ todos: [] }))
    .mockResolvedValueOnce(Response.json({ id: 1, title: 'Buy milk', done: false }, { status: 201 }));
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

const milk = { id: 1, title: 'Buy milk', done: true };
const bank = { id: 2, title: 'Call the bank', done: false };

test('ticks a todo from its box or its title, and unticks it', async () => {
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockResolvedValueOnce(Response.json({ todos: [bank, milk] }))
    .mockResolvedValueOnce(Response.json({ ...bank, done: true }))
    .mockResolvedValueOnce(Response.json({ ...milk, done: false }));
  vi.stubGlobal('fetch', fetchMock);
  render(TodosPage);
  const box = await screen.findByRole('checkbox', { name: 'Call the bank' });
  expect(box).not.toBeChecked();
  expect(screen.getByRole('checkbox', { name: 'Buy milk' })).toBeChecked();

  await fireEvent.click(box);
  expect(box).toBeChecked();
  await fireEvent.click(screen.getByText('Buy milk'));

  await waitFor(() => expect(screen.getByRole('checkbox', { name: 'Buy milk' })).not.toBeChecked());
  expect(box).toBeChecked();
  const patch = (body: string) => ({ method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body });
  expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/todos/2', patch('{"done":true}'));
  expect(fetchMock).toHaveBeenNthCalledWith(3, '/api/todos/1', patch('{"done":false}'));
});

test('puts the box back and says so when a tick fails', async () => {
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockResolvedValueOnce(Response.json({ todos: [bank, milk] }))
    .mockResolvedValueOnce(new Response(null, { status: 502 }))
    .mockRejectedValueOnce(new TypeError('Failed to fetch'));
  vi.stubGlobal('fetch', fetchMock);
  render(TodosPage);
  const [bankRow, milkRow] = within(await screen.findByRole('list')).getAllByRole('listitem');
  expect(within(bankRow).getByRole('status')).toHaveTextContent('');

  await fireEvent.click(within(bankRow).getByRole('checkbox'));
  await fireEvent.click(within(milkRow).getByRole('checkbox'));

  expect(await within(bankRow).findByText('Tick failed')).toHaveAttribute('role', 'status');
  expect(await within(milkRow).findByText('Untick failed')).toHaveAttribute('role', 'status');
  expect(within(bankRow).getByRole('checkbox')).not.toBeChecked();
  expect(within(milkRow).getByRole('checkbox')).toBeChecked();
});

test('ticks a todo once at a time', async () => {
  const answer = Promise.withResolvers<Response>();
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockResolvedValueOnce(Response.json({ todos: [bank] }))
    .mockReturnValueOnce(answer.promise);
  vi.stubGlobal('fetch', fetchMock);
  render(TodosPage);
  const box = await screen.findByRole('checkbox', { name: 'Call the bank' });

  await fireEvent.click(box);
  expect(box).toHaveAttribute('aria-disabled', 'true');
  await fireEvent.click(box);
  expect(box).toBeChecked();
  answer.resolve(Response.json({ ...bank, done: true }));

  await waitFor(() => expect(box).toHaveAttribute('aria-disabled', 'false'));
  expect(box).toBeChecked();
  expect(fetchMock).toHaveBeenCalledTimes(2);
});
