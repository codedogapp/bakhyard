import { afterEach, expect, test, vi } from 'vitest';

import type { Todo } from './api';
import { Editor, focusField } from './editor.svelte';
import { TodosState } from './state.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.replaceChildren();
});

const milk: Todo = { id: 1, title: 'Buy milk', done: true, priority: 'normal' };
const bank: Todo = { id: 2, title: 'Call the bank', done: false, priority: 'normal' };

/** An editor over the list, open on its first todo, with the draft typed. */
function editing(todos: Todo[], draft: string): { list: TodosState; editor: Editor } {
  const list = new TodosState();
  list.todos = todos;
  const editor = new Editor(list, () => undefined);
  editor.open(todos[0]);
  editor.draft = draft;
  return { list, editor };
}

function serve(answer: Response | Promise<Response>) {
  const fetchMock = vi.fn<typeof fetch>().mockReturnValue(Promise.resolve(answer));
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

function button(): HTMLButtonElement {
  return document.body.appendChild(document.createElement('button'));
}

test('opens on one todo at a time, with its title, and drops the other draft', () => {
  const { editor } = editing([bank, milk], 'Call the bank today');
  editor.status = { kind: 'saveFailed' };

  editor.open(milk);

  expect(editor.id).toBe(1);
  expect(editor.draft).toBe('Buy milk');
  expect(editor.status).toEqual({ kind: 'idle' });
});

test('saves the draft as typed, takes the new title and closes', async () => {
  const fetchMock = serve(Response.json({ ...bank, title: 'Call the bank today' }));
  // A tick that the service stored after the rename has answered first, so the rename's answer has the old done mark.
  const { list, editor } = editing([{ ...bank, done: true }, milk], ' Call the bank today');

  await editor.save();

  expect(list.todos).toEqual([{ ...bank, title: 'Call the bank today', done: true }, milk]);
  expect(editor.id).toBeUndefined();
  expect(editor.status).toEqual({ kind: 'idle' });
  expect(fetchMock.mock.calls[0][0]).toBe('/api/todos/2');
  expect(fetchMock.mock.calls[0][1]?.body).toBe('{"title":" Call the bank today"}');
});

test('closes with no request when the title is left as it was', async () => {
  const fetchMock = serve(Response.json(bank));
  const { editor } = editing([bank, milk], 'Call the bank');

  await editor.save();

  expect(editor.id).toBeUndefined();
  expect(fetchMock).not.toHaveBeenCalled();
});

test('sends a title that differs only in its spaces, which the service trims', async () => {
  const fetchMock = serve(Response.json(bank));
  const { list, editor } = editing([bank, milk], 'Call the bank ');

  await editor.save();

  expect(fetchMock.mock.calls[0][1]?.body).toBe('{"title":"Call the bank "}');
  expect(editor.id).toBeUndefined();
  expect(list.todos).toEqual([bank, milk]);
});

test('does nothing with a blank draft', async () => {
  const fetchMock = serve(Response.json(bank));
  const { editor } = editing([bank, milk], ' \t ');

  await editor.save();

  expect(editor.id).toBe(2);
  expect(editor.status).toEqual({ kind: 'idle' });
  expect(fetchMock).not.toHaveBeenCalled();
});

test('keeps the editor and the draft, with the reason, when the service refuses the title', async () => {
  serve(new Response('invalid title: has a control character\n', { status: 400 }));
  const { list, editor } = editing([bank, milk], 'Call\tthe bank');

  await editor.save();

  expect(editor.status).toEqual({ kind: 'refused', reason: 'invalid title: has a control character' });
  expect([editor.id, editor.draft]).toEqual([2, 'Call\tthe bank']);
  expect(list.todos).toEqual([bank, milk]);
});

test('says the save failed, keeps the draft, and saves on the next try', async () => {
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockRejectedValueOnce(new TypeError('Failed to fetch'))
    .mockResolvedValueOnce(Response.json({ ...bank, title: 'Call the bank today' }));
  vi.stubGlobal('fetch', fetchMock);
  const { list, editor } = editing([bank, milk], 'Call the bank today');

  await editor.save();
  expect(editor.status).toEqual({ kind: 'saveFailed' });
  expect([editor.id, editor.draft]).toEqual([2, 'Call the bank today']);
  await editor.save();

  expect(list.todos).toEqual([{ ...bank, title: 'Call the bank today' }, milk]);
  expect(editor.id).toBeUndefined();
});

test('changes nothing else while a save runs', async () => {
  const answer = Promise.withResolvers<Response>();
  const fetchMock = serve(answer.promise);
  const { editor } = editing([bank, milk], 'Call the bank today');

  const save = editor.save();
  expect(editor.busy).toBe(true);
  editor.open(milk);
  editor.close();
  editor.keydown(new KeyboardEvent('keydown', { key: 'Escape' }));
  await editor.save();
  expect([editor.id, editor.draft]).toEqual([2, 'Call the bank today']);
  answer.resolve(Response.json({ ...bank, title: 'Call the bank today' }));
  await save;

  expect(editor.busy).toBe(false);
  expect(fetchMock).toHaveBeenCalledOnce();
});

test.each([
  { key: 'Escape', isComposing: false, closes: true },
  { key: 'Escape', isComposing: true, closes: false },
  { key: 'Enter', isComposing: false, closes: false },
])('closes on the key $key, composing $isComposing: $closes', ({ key, isComposing, closes }) => {
  const { editor } = editing([bank, milk], 'Call the bank today');

  editor.keydown(new KeyboardEvent('keydown', { key, isComposing }));

  expect(editor.id).toBe(closes ? undefined : 2);
});

test('gives the focus back to the Edit button of the todo whose editor closed', () => {
  const { editor } = editing([bank, milk], 'Call the bank today');
  const [bankEdit, milkEdit] = [button(), button()];
  editor.editButton(2)(bankEdit);
  editor.editButton(1)(milkEdit);

  editor.close();

  expect(document.activeElement).toBe(bankEdit);
});

test('leaves the focus where the user moved it while the save ran', async () => {
  const answer = Promise.withResolvers<Response>();
  serve(answer.promise);
  const { editor } = editing([bank, milk], 'Call the bank today');
  editor.editButton(2)(button());
  const elsewhere = button();

  const save = editor.save();
  elsewhere.focus();
  answer.resolve(Response.json({ ...bank, title: 'Call the bank today' }));
  await save;

  expect(editor.id).toBeUndefined();
  expect(document.activeElement).toBe(elsewhere);
});

test('focuses the field with the caret after the title', () => {
  const input = document.body.appendChild(document.createElement('input'));
  input.value = 'Call the bank';
  input.setSelectionRange(0, 4);

  focusField(input);

  expect(document.activeElement).toBe(input);
  expect([input.selectionStart, input.selectionEnd]).toEqual([13, 13]);
});
