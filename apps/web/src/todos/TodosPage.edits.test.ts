import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import type { Todo } from './api';
import TodosPage from './TodosPage.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

/** Stubs fetch: the todos' calls go to `todos`, the tickets' call gets the answer without Jira, and none is dismissed. */
function stubFetch(todos: typeof fetch) {
  vi.stubGlobal(
    'fetch',
    vi.fn<typeof fetch>((...args) => {
      switch (args[0]) {
        case '/api/todos/tickets':
          return Promise.resolve(new Response(null, { status: 404 }));
        case '/api/todos/dismissals':
          return Promise.resolve(Response.json({ dismissals: [] }));
        default:
          return todos(...args);
      }
    })
  );
}

const milk: Todo = { id: 1, title: 'Buy milk', done: true, priority: 'normal' };
const bank: Todo = { id: 2, title: 'Call the bank', done: false, priority: 'normal' };

test('ticks a todo from its box or its title, and unticks it', async () => {
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockResolvedValueOnce(Response.json({ todos: [bank, milk] }))
    .mockResolvedValueOnce(Response.json({ ...bank, done: true }))
    .mockResolvedValueOnce(Response.json({ ...milk, done: false }));
  stubFetch(fetchMock);
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
  stubFetch(fetchMock);
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
  stubFetch(fetchMock);
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

test('renames a todo in its row, then gives the focus back to its Edit button', async () => {
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockResolvedValueOnce(Response.json({ todos: [bank, milk] }))
    .mockResolvedValueOnce(Response.json({ ...bank, title: 'Call the bank today' }));
  stubFetch(fetchMock);
  render(TodosPage);

  // A click that the browser dispatches at once: the field has the focus before the click returns.
  (await screen.findByRole('button', { name: 'Edit Call the bank' })).click();
  const field = screen.getByRole<HTMLInputElement>('textbox', { name: 'Title' });
  expect(field).toHaveFocus();
  expect([field.selectionStart, field.selectionEnd]).toEqual([13, 13]);
  expect(screen.queryByRole('checkbox', { name: 'Call the bank' })).not.toBeInTheDocument();
  await fireEvent.input(field, { target: { value: 'Call the bank today' } });
  await fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  await waitFor(() => expect(screen.getByRole('button', { name: 'Edit Call the bank today' })).toHaveFocus());
  expect(screen.getByRole('checkbox', { name: 'Call the bank today' })).not.toBeChecked();
  expect(screen.queryByRole('textbox', { name: 'Title' })).not.toBeInTheDocument();
  expect(fetchMock).toHaveBeenLastCalledWith('/api/todos/2', expect.objectContaining({ method: 'PATCH' }));
});

test('closes the editor on Cancel or Escape, with no request', async () => {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValueOnce(Response.json({ todos: [bank, milk] }));
  stubFetch(fetchMock);
  render(TodosPage);

  await fireEvent.click(await screen.findByRole('button', { name: 'Edit Buy milk' }));
  await fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
  await waitFor(() => expect(screen.getByRole('button', { name: 'Edit Buy milk' })).toHaveFocus());
  await fireEvent.click(screen.getByRole('button', { name: 'Edit Buy milk' }));
  await fireEvent.keyDown(screen.getByRole('textbox', { name: 'Title' }), { key: 'Escape' });

  await waitFor(() => expect(screen.getByRole('button', { name: 'Edit Buy milk' })).toHaveFocus());
  expect(screen.getByRole('checkbox', { name: 'Buy milk' })).toBeChecked();
  expect(fetchMock).toHaveBeenCalledOnce();
});

test('shows the save running, then the reason the service refused the title for', async () => {
  const answer = Promise.withResolvers<Response>();
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockResolvedValueOnce(Response.json({ todos: [bank, milk] }))
    .mockReturnValueOnce(answer.promise);
  stubFetch(fetchMock);
  render(TodosPage);

  await fireEvent.click(await screen.findByRole('button', { name: 'Edit Call the bank' }));
  await fireEvent.input(screen.getByRole('textbox', { name: 'Title' }), { target: { value: 'Call\tthe bank' } });
  await fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  expect(screen.getByRole('textbox', { name: 'Title' })).toHaveAttribute('readonly');
  for (const name of ['Save', 'Cancel', 'Edit Buy milk']) {
    expect(screen.getByRole('button', { name })).toHaveAttribute('aria-disabled', 'true');
  }
  expect(screen.getByText('Saving…')).toHaveAttribute('role', 'status');
  answer.resolve(new Response('invalid title: has a control character\n', { status: 400 }));

  expect(await screen.findByText('invalid title: has a control character')).toHaveAttribute('role', 'status');
  expect(screen.getByRole('textbox', { name: 'Title' })).not.toHaveAttribute('readonly');
  expect(screen.getByRole('button', { name: 'Edit Buy milk' })).toHaveAttribute('aria-disabled', 'false');
});

test('deletes a todo from its editor, then gives the focus to the Add field once none is left', async () => {
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockResolvedValueOnce(Response.json({ todos: [bank] }))
    .mockResolvedValueOnce(new Response(null, { status: 204 }));
  stubFetch(fetchMock);
  render(TodosPage);

  await fireEvent.click(await screen.findByRole('button', { name: 'Edit Call the bank' }));
  await fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
  const dialog = screen.getByRole('alertdialog', { name: 'Delete this todo?' });
  await fireEvent.click(within(dialog).getByRole('button', { name: 'Delete' }));

  await waitFor(() => expect(screen.getByRole('textbox', { name: 'New todo' })).toHaveFocus());
  expect(screen.getByText('No todos on the list.')).toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
  expect(fetchMock).toHaveBeenLastCalledWith('/api/todos/2', { method: 'DELETE' });
});
