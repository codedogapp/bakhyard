import { flushSync } from 'svelte';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import type { Todo } from './api';
import DeleteDialog from './DeleteDialog.svelte';
import { Editor } from './editor.svelte';
import { TodosState } from './state.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

const bank: Todo = { id: 2, title: 'Call the bank', done: false, priority: 'normal' };

/** The dialog, open and asking about the todo whose editor is open, with a new title typed. */
function asking(): Editor {
  const list = new TodosState();
  list.todos = [bank];
  const editor = new Editor(list, () => undefined);
  render(DeleteDialog, { editor });
  editor.open(bank);
  editor.draft = 'Call the bank today';
  flushSync();
  editor.confirmDelete();
  return editor;
}

test('names the todo by its title as the list has it, not as typed', () => {
  asking();

  const dialog = screen.getByRole('alertdialog', { name: 'Delete this todo?' });
  expect(dialog).toHaveAccessibleDescription("“Call the bank” will be deleted. This can't be undone.");
  expect(
    within(dialog)
      .getAllByRole('button')
      .map(({ textContent }) => textContent)
  ).toEqual(['Cancel', 'Delete']);
});

test('closes on Cancel with no request', async () => {
  const fetchMock = vi.fn<typeof fetch>();
  vi.stubGlobal('fetch', fetchMock);
  const editor = asking();

  await fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

  expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  expect(editor.id).toBe(2);
  expect(fetchMock).not.toHaveBeenCalled();
});

test('closes and deletes the todo on Delete', async () => {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 204 }));
  vi.stubGlobal('fetch', fetchMock);
  const editor = asking();

  await fireEvent.click(screen.getByRole('button', { name: 'Delete' }));

  expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  await waitFor(() => expect(editor.id).toBeUndefined());
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos/2', { method: 'DELETE' });
});
