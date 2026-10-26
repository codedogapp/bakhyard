import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import AddForm from './AddForm.svelte';
import { TodosState } from './state.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

const milk = { id: 1, title: 'Buy milk', done: false };
const bank = { id: 2, title: 'Call the bank', done: false };

function renderForm(): TodosState {
  const list = new TodosState();
  list.todos = [milk];
  render(AddForm, { props: { list } });
  return list;
}

test('adds what the field holds, and empties it', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(Response.json(bank, { status: 201 })));
  const list = renderForm();
  const field = screen.getByRole('textbox', { name: 'New todo' });
  expect(screen.getByRole('status').textContent).toBe('');

  await fireEvent.input(field, { target: { value: 'Call the bank' } });
  await fireEvent.click(screen.getByRole('button', { name: 'Add' }));

  expect(await screen.findByRole('status')).toHaveTextContent('Added');
  expect(field).toHaveValue('');
  expect(list.todos).toEqual([bank, milk]);
});

test('is read-only while it adds, and keeps the text when the add fails', async () => {
  const answer = Promise.withResolvers<Response>();
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockReturnValue(answer.promise));
  renderForm();
  const field = screen.getByRole('textbox', { name: 'New todo' });
  const button = screen.getByRole('button', { name: 'Add' });

  await fireEvent.input(field, { target: { value: 'Call the bank' } });
  await fireEvent.click(button);

  expect(screen.getByRole('status')).toHaveTextContent('Adding…');
  expect(field).toHaveAttribute('readonly');
  expect(button).toHaveAttribute('aria-disabled', 'true');

  answer.resolve(new Response('Internal Server Error\n', { status: 500 }));

  await vi.waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('Add failed'));
  expect(field).not.toHaveAttribute('readonly');
  expect(button).toHaveAttribute('aria-disabled', 'false');
  expect(field).toHaveValue('Call the bank');
});

test("shows the service's reason for refusing a title", async () => {
  const refusal = new Response('invalid title: has a control character\n', { status: 400 });
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(refusal));
  renderForm();

  await fireEvent.input(screen.getByRole('textbox', { name: 'New todo' }), { target: { value: 'Call\tthe bank' } });
  await fireEvent.click(screen.getByRole('button', { name: 'Add' }));

  await vi.waitFor(() =>
    expect(screen.getByRole('status')).toHaveTextContent('invalid title: has a control character')
  );
});
