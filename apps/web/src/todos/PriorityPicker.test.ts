import { flushSync } from 'svelte';
import { fireEvent, render, screen, within } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import type { Todo } from './api';
import { Editor } from './editor.svelte';
import { Focus } from './focus';
import PriorityPicker from './PriorityPicker.svelte';
import { TodosState } from './state.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

const bank: Todo = { id: 2, title: 'Call the bank', done: false, priority: 'low' };

/** The priorities of an editor open on the todo. */
function picking(): Editor {
  const list = new TodosState();
  list.todos = [bank];
  const editor = new Editor(list, new Focus(() => undefined));
  editor.open(bank);
  render(PriorityPicker, { editor });
  return editor;
}

test("offers the priorities in the page's order, as a group named Priority, with the todo's checked", () => {
  picking();

  const group = screen.getByRole('group', { name: 'Priority' });
  const radios = within(group).getAllByRole('radio');
  expect(radios.map((radio) => radio.closest('label')?.textContent?.trim())).toEqual(['High', 'Normal', 'Low']);
  expect(within(group).getByRole('radio', { checked: true })).toHaveAccessibleName('Low');
  expect(new Set(radios.map((radio) => radio.getAttribute('name')))).toEqual(new Set(['priority']));
});

test('takes the priority picked as the new one', async () => {
  const editor = picking();

  await fireEvent.click(screen.getByRole('radio', { name: 'High' }));

  expect(editor.draft.priority).toBe('high');
  expect(screen.getByRole('radio', { name: 'High' })).toBeChecked();
  expect(screen.getByRole('radio', { name: 'Low' })).not.toBeChecked();
});

test('keeps the priority while a save runs, and takes a pick again once it has failed', async () => {
  const answer = Promise.withResolvers<Response>();
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockReturnValue(answer.promise));
  const editor = picking();
  editor.draft.title = 'Call the bank today';
  const high = screen.getByRole('radio', { name: 'High' });

  const save = editor.save();
  flushSync();
  expect(high).toHaveAttribute('aria-disabled', 'true');
  await fireEvent.click(high);
  expect(editor.draft.priority).toBe('low');
  expect(high).not.toBeChecked();
  answer.resolve(new Response(null, { status: 502 }));
  await save;
  flushSync();

  expect(high).toHaveAttribute('aria-disabled', 'false');
  await fireEvent.click(high);
  expect(editor.draft.priority).toBe('high');
});
