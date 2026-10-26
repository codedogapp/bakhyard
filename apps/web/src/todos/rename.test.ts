import { afterEach, expect, test, vi } from 'vitest';

import type { Todo } from './api';
import { focusField, Rename } from './rename.svelte';
import { TodosState } from './state.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.replaceChildren();
});

const milk = { id: 1, title: 'Buy milk', done: true };
const bank = { id: 2, title: 'Call the bank', done: false };

/** A rename editor over the list, open on its first todo, with the draft typed. */
function editing(todos: Todo[], draft: string): { list: TodosState; rename: Rename } {
  const list = new TodosState();
  list.todos = todos;
  const rename = new Rename(list);
  rename.open(todos[0]);
  rename.draft = draft;
  return { list, rename };
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
  const { rename } = editing([bank, milk], 'Call the bank today');
  rename.status = { kind: 'failed' };

  rename.open(milk);

  expect(rename.id).toBe(1);
  expect(rename.draft).toBe('Buy milk');
  expect(rename.status).toEqual({ kind: 'idle' });
});

test('saves the draft as typed, takes the new title and closes', async () => {
  const fetchMock = serve(Response.json({ ...bank, title: 'Call the bank today' }));
  // A tick that the service stored after the rename has answered first, so the rename's answer has the old done mark.
  const { list, rename } = editing([{ ...bank, done: true }, milk], ' Call the bank today');

  await rename.save();

  expect(list.todos).toEqual([{ ...bank, title: 'Call the bank today', done: true }, milk]);
  expect(rename.id).toBeUndefined();
  expect(rename.status).toEqual({ kind: 'idle' });
  expect(fetchMock.mock.calls[0][0]).toBe('/api/todos/2');
  expect(fetchMock.mock.calls[0][1]?.body).toBe('{"title":" Call the bank today"}');
});

test('closes with no request when the title is left as it was', async () => {
  const fetchMock = serve(Response.json(bank));
  const { rename } = editing([bank, milk], 'Call the bank');

  await rename.save();

  expect(rename.id).toBeUndefined();
  expect(fetchMock).not.toHaveBeenCalled();
});

test('sends a title that differs only in its spaces, which the service trims', async () => {
  const fetchMock = serve(Response.json(bank));
  const { list, rename } = editing([bank, milk], 'Call the bank ');

  await rename.save();

  expect(fetchMock.mock.calls[0][1]?.body).toBe('{"title":"Call the bank "}');
  expect(rename.id).toBeUndefined();
  expect(list.todos).toEqual([bank, milk]);
});

test('does nothing with a blank draft', async () => {
  const fetchMock = serve(Response.json(bank));
  const { rename } = editing([bank, milk], ' \t ');

  await rename.save();

  expect(rename.id).toBe(2);
  expect(rename.status).toEqual({ kind: 'idle' });
  expect(fetchMock).not.toHaveBeenCalled();
});

test('keeps the editor and the draft, with the reason, when the service refuses the title', async () => {
  serve(new Response('invalid title: has a control character\n', { status: 400 }));
  const { list, rename } = editing([bank, milk], 'Call\tthe bank');

  await rename.save();

  expect(rename.status).toEqual({ kind: 'refused', reason: 'invalid title: has a control character' });
  expect([rename.id, rename.draft]).toEqual([2, 'Call\tthe bank']);
  expect(list.todos).toEqual([bank, milk]);
});

test('says the save failed, keeps the draft, and saves on the next try', async () => {
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockRejectedValueOnce(new TypeError('Failed to fetch'))
    .mockResolvedValueOnce(Response.json({ ...bank, title: 'Call the bank today' }));
  vi.stubGlobal('fetch', fetchMock);
  const { list, rename } = editing([bank, milk], 'Call the bank today');

  await rename.save();
  expect(rename.status).toEqual({ kind: 'failed' });
  expect([rename.id, rename.draft]).toEqual([2, 'Call the bank today']);
  await rename.save();

  expect(list.todos).toEqual([{ ...bank, title: 'Call the bank today' }, milk]);
  expect(rename.id).toBeUndefined();
});

test('changes nothing else while a save runs', async () => {
  const answer = Promise.withResolvers<Response>();
  const fetchMock = serve(answer.promise);
  const { rename } = editing([bank, milk], 'Call the bank today');

  const save = rename.save();
  expect(rename.saving).toBe(true);
  rename.open(milk);
  rename.close();
  rename.keydown(new KeyboardEvent('keydown', { key: 'Escape' }));
  await rename.save();
  expect([rename.id, rename.draft]).toEqual([2, 'Call the bank today']);
  answer.resolve(Response.json({ ...bank, title: 'Call the bank today' }));
  await save;

  expect(rename.saving).toBe(false);
  expect(fetchMock).toHaveBeenCalledOnce();
});

test.each([
  { key: 'Escape', isComposing: false, closes: true },
  { key: 'Escape', isComposing: true, closes: false },
  { key: 'Enter', isComposing: false, closes: false },
])('closes on the key $key, composing $isComposing: $closes', ({ key, isComposing, closes }) => {
  const { rename } = editing([bank, milk], 'Call the bank today');

  rename.keydown(new KeyboardEvent('keydown', { key, isComposing }));

  expect(rename.id).toBe(closes ? undefined : 2);
});

test('gives the focus back to the Edit button of the editor that closed, once', () => {
  const { rename } = editing([bank, milk], 'Call the bank today');
  const [bankEdit, milkEdit] = [button(), button()];

  rename.close();
  rename.editButton(1)(milkEdit);
  expect(document.activeElement).toBe(document.body);
  rename.editButton(2)(bankEdit);
  expect(document.activeElement).toBe(bankEdit);
  bankEdit.blur();
  rename.editButton(2)(bankEdit);

  expect(document.activeElement).toBe(document.body);
});

test('focuses the field with the caret after the title', () => {
  const input = document.body.appendChild(document.createElement('input'));
  input.value = 'Call the bank';
  input.setSelectionRange(0, 4);

  focusField(input);

  expect(document.activeElement).toBe(input);
  expect([input.selectionStart, input.selectionEnd]).toEqual([13, 13]);
});
