import { afterEach, expect, test, vi } from 'vitest';

import type { Todo } from './api';
import { Editor } from './editor.svelte';
import { Focus } from './focus';
import { TodosState } from './state.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.replaceChildren();
});

const rent: Todo = { id: 3, title: 'Pay the rent', done: false, priority: 'normal' };
const bank: Todo = { id: 2, title: 'Call the bank', done: false, priority: 'normal' };
const milk: Todo = { id: 1, title: 'Buy milk', done: true, priority: 'normal' };

/** An editor over the list, with the page's dialog and the rows' Edit buttons, open on the todo with its Delete. */
function deleting(todos: Todo[], todo: Todo) {
  const list = new TodosState();
  list.todos = todos;
  const focusAdd = vi.fn<() => void>();
  const focus = new Focus(focusAdd);
  const editor = new Editor(list, focus);
  const dialog = document.body.appendChild(document.createElement('dialog'));
  editor.dialog(dialog);
  const edits = new Map(todos.map(({ id }) => [id, button()]));
  for (const [id, edit] of edits) {
    focus.editButton(id)(edit);
  }
  editor.open(todo);
  editor.draft = `${todo.title} today`;
  const deleteButton = button();
  editor.deleteButton(deleteButton);
  return { list, editor, focus, dialog, edits, deleteButton, focusAdd };
}

function serve(...answers: (Response | Promise<Response>)[]) {
  const fetchMock = vi.fn<typeof fetch>();
  for (const answer of answers) {
    fetchMock.mockReturnValueOnce(Promise.resolve(answer));
  }
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

function button(): HTMLButtonElement {
  return document.body.appendChild(document.createElement('button'));
}

const deleted = () => new Response(null, { status: 204 });

test('asks in the dialog before a delete, and closes it on Cancel with no request', () => {
  const fetchMock = serve();
  const { list, editor, dialog } = deleting([bank, milk], bank);

  editor.confirmDelete();
  expect(dialog.open).toBe(true);
  editor.cancelDelete();

  expect(dialog.open).toBe(false);
  expect([editor.id, editor.draft]).toEqual([2, 'Call the bank today']);
  expect(list.todos).toEqual([bank, milk]);
  expect(fetchMock).not.toHaveBeenCalled();
});

test.each([
  { todo: bank, left: [rent, milk], focus: 1 },
  { todo: milk, left: [rent, bank], focus: 2 },
  { todo: rent, left: [bank, milk], focus: 2 },
])(
  'deletes $todo.title, closes the editor, and focuses the Edit button of todo $focus',
  async ({ todo, left, focus }) => {
    const fetchMock = serve(deleted());
    const { list, editor, dialog, edits, focusAdd } = deleting([rent, bank, milk], todo);
    editor.confirmDelete();

    await editor.delete();

    expect(fetchMock).toHaveBeenCalledExactlyOnceWith(`/api/todos/${todo.id}`, { method: 'DELETE' });
    expect(list.todos).toEqual(left);
    expect([editor.id, editor.status, dialog.open]).toEqual([undefined, { kind: 'idle' }, false]);
    expect(document.activeElement).toBe(edits.get(focus));
    expect(focusAdd).not.toHaveBeenCalled();
  }
);

const urgentBank: Todo = { ...bank, priority: 'high' };

test.each([
  { todo: urgentBank, focus: 3 },
  { todo: rent, focus: 1 },
  { todo: milk, focus: 3 },
])('moves the focus on from $todo.title in the page order, High first, to todo $focus', async ({ todo, focus }) => {
  serve(deleted());
  const { editor, edits } = deleting([rent, urgentBank, milk], todo);

  await editor.delete();

  expect(document.activeElement).toBe(edits.get(focus));
});

test('gives the focus to the Add field once no todo is left', async () => {
  serve(deleted());
  const { list, editor, focusAdd } = deleting([bank], bank);

  await editor.delete();

  expect(list.todos).toEqual([]);
  expect(focusAdd).toHaveBeenCalledOnce();
});

test('says the delete failed, keeps the editor and the draft, and deletes on the next try', async () => {
  const fetchMock = serve(new Response(null, { status: 500 }), deleted());
  const { list, editor, edits } = deleting([bank, milk], bank);

  await editor.delete();
  expect(editor.status).toEqual({ kind: 'deleteFailed' });
  expect([editor.id, editor.draft]).toEqual([2, 'Call the bank today']);
  expect(list.todos).toEqual([bank, milk]);
  await editor.delete();

  expect(list.todos).toEqual([milk]);
  expect(document.activeElement).toBe(edits.get(1));
  expect(fetchMock).toHaveBeenCalledTimes(2);
});

test('changes nothing else while a delete runs', async () => {
  const answer = Promise.withResolvers<Response>();
  const fetchMock = serve(answer.promise);
  const { list, editor, dialog } = deleting([bank, milk], bank);

  const deleteRun = editor.delete();
  expect(editor.busy).toBe(true);
  editor.open(milk);
  editor.close();
  editor.keydown(new KeyboardEvent('keydown', { key: 'Escape' }));
  editor.confirmDelete();
  expect(dialog.open).toBe(false);
  await Promise.all([editor.save(), editor.delete()]);
  expect([editor.id, editor.draft, list.todos]).toEqual([2, 'Call the bank today', [bank, milk]]);
  answer.resolve(deleted());
  await deleteRun;

  expect(editor.busy).toBe(false);
  expect(list.todos).toEqual([milk]);
  expect(fetchMock).toHaveBeenCalledOnce();
});

test('moves the focus on from the closed dialog, where WebKit leaves it after a click', async () => {
  serve(deleted());
  const { editor, dialog, edits } = deleting([bank, milk], bank);
  editor.confirmDelete();
  dialog.appendChild(document.createElement('button')).focus();

  await editor.delete();

  expect(document.activeElement).toBe(edits.get(1));
});

test('gives the focus back to Delete as the dialog closes, from the dialog, where WebKit leaves it after a click', () => {
  const { editor, dialog, deleteButton } = deleting([bank, milk], bank);
  editor.confirmDelete();
  dialog.appendChild(document.createElement('button')).focus();

  editor.cancelDelete();
  dialog.dispatchEvent(new Event('close'));

  expect(document.activeElement).toBe(deleteButton);
});

test('leaves the focus where the browser gave it back as the dialog closed', () => {
  const { dialog, edits } = deleting([bank, milk], bank);
  edits.get(1)?.focus();

  dialog.dispatchEvent(new Event('close'));

  expect(document.activeElement).toBe(edits.get(1));
});

test('leaves the focus where the user moved it while the delete ran', async () => {
  const answer = Promise.withResolvers<Response>();
  serve(answer.promise);
  const { editor, focusAdd } = deleting([bank], bank);
  const elsewhere = button();

  const deleteRun = editor.delete();
  elsewhere.focus();
  answer.resolve(deleted());
  await deleteRun;

  expect(document.activeElement).toBe(elsewhere);
  expect(focusAdd).not.toHaveBeenCalled();
});

test('gives the focus to a todo added while the delete ran', async () => {
  const answer = Promise.withResolvers<Response>();
  serve(answer.promise);
  const { list, editor, focus, focusAdd } = deleting([bank], bank);

  const deleteRun = editor.delete();
  list.todos = [milk, bank];
  const milkEdit = button();
  focus.editButton(1)(milkEdit);
  answer.resolve(deleted());
  await deleteRun;

  expect(list.todos).toEqual([milk]);
  expect(document.activeElement).toBe(milkEdit);
  expect(focusAdd).not.toHaveBeenCalled();
});
