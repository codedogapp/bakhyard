import { flushSync, tick } from 'svelte';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import type { Ticket } from '../jira/tickets';
import type { Todo } from './api';
import { Editor } from './editor.svelte';
import { Focus } from './focus';
import { TodosState } from './state.svelte';
import TicketPicker from './TicketPicker.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

const bank: Todo = { id: 2, title: 'Call the bank', done: false, priority: 'normal' };

function ticket(key: string, summary: string): Ticket {
  return { key, summary, status: 'To Do', category: 'to-do', url: `https://jira.example.com/browse/${key}` };
}

/** My open tickets, in Jira's order. */
const open = new Map(
  [ticket('WEB-12', 'Show the order history'), ticket('WEB-9', 'Fix the date picker')].map((each) => [each.key, each])
);

/** The ticket picker of an editor open on the todo, with my open tickets. */
function picking(todo: Todo, tickets: ReadonlyMap<string, Ticket> = open) {
  const list = new TodosState();
  list.todos = [todo];
  const editor = new Editor(list, new Focus(() => undefined));
  editor.open(todo);
  const { rerender } = render(TicketPicker, { editor, open: tickets });
  return { editor, rerender };
}

const select = () => screen.getByRole<HTMLSelectElement>('combobox', { name: 'Ticket' });
const options = () => [...select().options].map(({ value, text }) => [value, text]);

test('offers no ticket, then my open tickets in their order with their summaries, and picks none for a todo without one', () => {
  picking(bank);

  expect(options()).toEqual([
    ['', 'No ticket'],
    ['WEB-12', 'WEB-12 · Show the order history'],
    ['WEB-9', 'WEB-9 · Fix the date picker'],
  ]);
  expect(select()).toHaveValue('');
});

test.each([
  ['one of mine', 'WEB-9', ['', 'WEB-12', 'WEB-9']],
  ["by its key alone, when it isn't one of mine", 'WEB-4', ['', 'WEB-4', 'WEB-12', 'WEB-9']],
])("picks the todo's own ticket, %s", (_, key, keys) => {
  picking({ ...bank, ticket: key });

  expect(options().map(([value]) => value)).toEqual(keys);
  expect(select()).toHaveValue(key);
});

test("keeps offering the todo's own ticket after another pick, to pick it back", async () => {
  const { editor } = picking({ ...bank, ticket: 'WEB-4' });

  await fireEvent.change(select(), { target: { value: '' } });
  expect(options().map(([value]) => value)).toEqual(['', 'WEB-4', 'WEB-12', 'WEB-9']);
  await fireEvent.change(select(), { target: { value: 'WEB-4' } });

  expect(editor.draft.ticket).toBe('WEB-4');
});

test("takes the ticket picked as the draft's, and shows the draft's", async () => {
  const { editor } = picking(bank);

  await fireEvent.change(select(), { target: { value: 'WEB-9' } });
  expect(editor.draft.ticket).toBe('WEB-9');
  editor.draft.ticket = 'WEB-12';
  flushSync();

  expect(select()).toHaveValue('WEB-12');
});

test('shows nothing to pick without open tickets, for a todo without one', () => {
  picking(bank, new Map());

  expect(screen.queryByRole('combobox')).toBeNull();
});

test("offers the todo's own ticket to unlink, without open tickets", () => {
  picking({ ...bank, ticket: 'WEB-4' }, new Map());

  expect(options()).toEqual([
    ['', 'No ticket'],
    ['WEB-4', 'WEB-4'],
  ]);
});

test('keeps the pick when my open tickets arrive after the editor opened', async () => {
  const { rerender } = picking({ ...bank, ticket: 'WEB-9' }, new Map());

  await fireEvent.change(select(), { target: { value: '' } });
  await rerender({ open });
  await tick();
  expect(options().map(([value]) => value)).toEqual(['', 'WEB-12', 'WEB-9']);
  expect(select()).toHaveValue('');
  await fireEvent.change(select(), { target: { value: 'WEB-9' } });
  await rerender({ open: new Map([...open].toReversed()) });
  await tick();

  expect(options().map(([value]) => value)).toEqual(['', 'WEB-9', 'WEB-12']);
  expect(select()).toHaveValue('WEB-9');
});

test('disables the ticket while a save runs, and takes a pick again once it has failed', async () => {
  const answer = Promise.withResolvers<Response>();
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockReturnValue(answer.promise));
  const { editor } = picking(bank);
  editor.draft.title = 'Call the bank today';

  const save = editor.save();
  flushSync();
  expect(select()).toBeDisabled();
  answer.resolve(new Response(null, { status: 502 }));
  await save;
  flushSync();

  expect(select()).toBeEnabled();
  await fireEvent.change(select(), { target: { value: 'WEB-12' } });
  expect(editor.draft.ticket).toBe('WEB-12');
});
