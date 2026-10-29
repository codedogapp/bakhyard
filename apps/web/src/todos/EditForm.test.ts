import { flushSync } from 'svelte';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import type { Ticket } from '../jira/tickets';
import type { Todo } from './api';
import EditForm from './EditForm.svelte';
import { Editor } from './editor.svelte';
import { Focus } from './focus';
import { TodosState } from './state.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

const bank: Todo = { id: 2, title: 'Call the bank', done: false, priority: 'normal' };
const milk: Todo = { id: 1, title: 'Buy milk', done: false, priority: 'normal' };

/** The editor's form, open on the bank, the first of two todos in its group, with my open tickets. */
function editing(open: ReadonlyMap<string, Ticket> = new Map()): TodosState {
  const list = new TodosState();
  list.todos = [bank, milk];
  const editor = new Editor(list, new Focus(() => undefined));
  editor.open(bank);
  render(EditForm, { editor, open });
  return list;
}

const moveUp = () => screen.getByRole('button', { name: 'Move up' });
const moveDown = () => screen.getByRole('button', { name: 'Move down' });

test('offers the moves after the other buttons, the way past the group’s end muted', () => {
  editing();

  expect(screen.getAllByRole('button').map((button) => button.textContent?.trim())).toEqual([
    'Save',
    'Cancel',
    'Delete',
    'Move up',
    'Move down',
  ]);
  expect(moveUp()).toHaveAttribute('aria-disabled', 'true');
  expect(moveDown()).toHaveAttribute('aria-disabled', 'false');
  expect(moveDown()).toHaveAttribute('type', 'button');
});

test('offers my open tickets after the priority and before the buttons', () => {
  const url = 'https://jira.example.com/browse/WEB-9';
  editing(new Map([['WEB-9', { key: 'WEB-9', summary: 'Fix it', status: 'To Do', category: 'to-do', url }]]));

  const [low, ticket, save] = [
    screen.getByRole('radio', { name: 'Low' }),
    screen.getByRole('combobox', { name: 'Ticket' }),
    screen.getByRole('button', { name: 'Save' }),
  ];
  expect(low.compareDocumentPosition(ticket)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  expect(ticket.compareDocumentPosition(save)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  expect(screen.getByRole('option', { name: 'WEB-9 · Fix it' })).toBeInTheDocument();
});

test('moves the todo down a step, then says so, and turns the muted end around', async () => {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(Response.json(bank));
  vi.stubGlobal('fetch', fetchMock);
  const list = editing();

  await fireEvent.click(moveDown());
  await vi.waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('Moved down'));

  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos/2', expect.objectContaining({ body: '{"after":1}' }));
  expect(list.todos?.map(({ id }) => id)).toEqual([1, 2]);
  expect(moveUp()).toHaveAttribute('aria-disabled', 'false');
  expect(moveDown()).toHaveAttribute('aria-disabled', 'true');
});

test('mutes every button while the move runs, and moves the todo back up after it', async () => {
  const answer = Promise.withResolvers<Response>();
  const fetchMock = vi.fn<typeof fetch>().mockReturnValueOnce(answer.promise).mockResolvedValue(Response.json(bank));
  vi.stubGlobal('fetch', fetchMock);
  const list = editing();

  await fireEvent.click(moveDown());
  flushSync();
  expect(screen.getByRole('status')).toHaveTextContent('Moving…');
  for (const button of screen.getAllByRole('button')) {
    expect(button).toHaveAttribute('aria-disabled', 'true');
  }
  answer.resolve(Response.json(bank));
  await vi.waitFor(() => expect(moveUp()).toHaveAttribute('aria-disabled', 'false'));

  await fireEvent.click(moveUp());
  await vi.waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('Moved up'));

  expect(fetchMock).toHaveBeenLastCalledWith('/api/todos/2', expect.objectContaining({ body: '{"before":1}' }));
  expect(list.todos?.map(({ id }) => id)).toEqual([2, 1]);
});
