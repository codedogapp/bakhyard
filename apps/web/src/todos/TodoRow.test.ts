import { flushSync } from 'svelte';
import { render, within } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import type { Todo } from './api';
import { Drag } from './drag.svelte';
import { Editor } from './editor.svelte';
import { Focus } from './focus';
import { TodosState } from './state.svelte';
import TodoRow from './TodoRow.svelte';
import type { Ticket } from './tickets';

afterEach(() => {
  document.body.replaceChildren();
});

const bank: Todo = { id: 2, title: 'Call the bank', done: false, priority: 'normal' };
const milk: Todo = { id: 1, title: 'Buy milk', done: false, priority: 'normal' };

/**
 * The bank's row 100 px down the page and the milk's under it, each 50 px tall, in one list, with my open tickets and
 * the done tickets that the todos name.
 */
function rows(
  open: ReadonlyMap<string, Ticket> = new Map(),
  todos = [bank, milk],
  done: ReadonlyMap<string, Ticket> = new Map()
) {
  const list = new TodosState();
  list.todos = todos;
  const focus = new Focus(() => undefined);
  const editor = new Editor(list, focus);
  const drag = new Drag(list, editor);
  const target = document.body.appendChild(document.createElement('ul'));
  for (const todo of list.todos) {
    render(TodoRow, { target, props: { list, todo, editor, focus, drag, open, done } });
  }
  flushSync();
  const [first, second] = target.querySelectorAll('li');
  if (!(first && second)) {
    throw new Error('no rows');
  }
  first.getBoundingClientRect = () => new DOMRect(0, 100, 300, 50);
  second.getBoundingClientRect = () => new DOMRect(0, 150, 300, 50);
  return { list, editor, first, second };
}

const grip = (row: HTMLElement) => row.querySelector<HTMLElement>('.grip');

function pointer(type: string, clientY: number): PointerEvent {
  return new PointerEvent(type, { pointerId: 1, isPrimary: true, clientY });
}

test('ends each row with a grip that only a pointer can use', () => {
  const { first } = rows();

  expect(first.lastElementChild).toBe(grip(first));
  expect(grip(first)).toHaveAttribute('aria-hidden', 'true');
  expect(grip(first)).not.toHaveAttribute('tabindex');
  expect(
    within(first)
      .getAllByRole('button')
      .map((button) => button.textContent?.trim())
  ).toEqual(['Edit']);
});

test("takes the grip away while the row's editor is open", () => {
  const { editor, first, second } = rows();

  editor.open(bank);
  flushSync();

  expect(grip(first)).toBeNull();
  expect(grip(second)).not.toBeNull();
});

test('lifts the dragged row in place and marks where the todo would land', () => {
  const { first, second } = rows();
  const handle = grip(first) as HTMLElement;
  handle.setPointerCapture = vi.fn<(pointerId: number) => void>();

  handle.dispatchEvent(pointer('pointerdown', 110));
  handle.dispatchEvent(pointer('pointermove', 190));
  flushSync();

  expect(first).toHaveClass('lifted');
  expect(second).toHaveClass('drop-after');

  handle.dispatchEvent(new PointerEvent('lostpointercapture', { pointerId: 1 }));
  flushSync();

  expect(first).not.toHaveClass('lifted');
  expect(second).not.toHaveClass('drop-after');
});

test('marks the grips as waiting while a drop runs', () => {
  const { list, first, second } = rows();

  list.dropping = true;
  flushSync();

  expect(grip(first)).toHaveClass('waiting');
  expect(grip(second)).toHaveClass('waiting');
  list.dropping = false;
  flushSync();
  expect(grip(first)).not.toHaveClass('waiting');
});

test('says the move failed on the row whose drop failed', () => {
  const { list, first, second } = rows();

  list.dropFailed = milk.id;
  flushSync();

  expect(within(second).getByRole('status')).toHaveTextContent('Move failed');
  expect(within(first).getByRole('status').textContent).toBe('');
});

const picker: Ticket = {
  key: 'WEB-9',
  summary: 'Fix the date picker in Safari',
  status: 'In Progress',
  category: 'in-progress',
  url: 'https://jira.example.com/browse/WEB-9',
};
const linked: Todo = { ...bank, ticket: 'WEB-9' };

test("shows the ticket a todo names under its title, in the box's label, with its status when it's open", () => {
  const { first, second } = rows(new Map([['WEB-9', picker]]), [linked, milk]);

  const box = within(first).getByRole('checkbox', { name: 'Call the bank WEB-9 In Progress' });
  const label = box.closest('label');
  expect(label?.querySelector('.title')).toHaveTextContent(/^Call the bank$/);
  expect(label?.querySelector('.badge')).toHaveTextContent('WEB-9 In Progress');
  expect(within(first).getByRole('button', { name: 'Edit Call the bank' })).toBeInTheDocument();
  expect(within(second).getByRole('checkbox', { name: 'Buy milk' })).toBeInTheDocument();
  expect(second.querySelector('.badge')).toBeNull();
});

test("shows the key alone when the ticket isn't one of my open tickets", () => {
  const { first } = rows(new Map([['OPS-31', { ...picker, key: 'OPS-31' }]]), [linked, milk]);

  expect(within(first).getByRole('checkbox', { name: 'Call the bank WEB-9' })).toBeInTheDocument();
  expect(first.querySelector('.chip')).toBeNull();
});

const shipped: Ticket = { ...picker, status: 'Closed', category: 'done' };

test("shows a done ticket's status and offers to tick its todo, in the box's label", () => {
  const { first } = rows(new Map(), [linked, milk], new Map([['WEB-9', shipped]]));

  const box = within(first).getByRole('checkbox', { name: 'Call the bank WEB-9 Closed Done in Jira' });
  expect(box).not.toBeChecked();
  expect(box.closest('label')?.querySelector('.badge')).toHaveTextContent('WEB-9 Closed Done in Jira');
  expect(
    within(first)
      .getAllByRole('button')
      .map((button) => button.textContent?.trim())
  ).toEqual(['Edit']);
});

test('makes no offer once the todo is ticked', () => {
  const { first } = rows(new Map(), [{ ...linked, done: true }, milk], new Map([['WEB-9', shipped]]));

  expect(within(first).getByRole('checkbox', { name: 'Call the bank WEB-9 Closed' })).toBeChecked();
  expect(first.querySelector('.offer')).toBeNull();
});
