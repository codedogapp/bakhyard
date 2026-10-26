import { flushSync } from 'svelte';
import { afterEach, expect, test, vi } from 'vitest';

import type { Priority, Todo } from './api';
import { Editor } from './editor.svelte';
import { Focus } from './focus';
import { TodosState } from './state.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.replaceChildren();
});

function todo(id: number, priority: Priority): Todo {
  return { id, title: `Todo ${id}`, done: false, priority };
}

// High 6, 4 and 2; Normal 5 and 3; Low 1.
const [six, five, four, three, two, one] = [
  todo(6, 'high'),
  todo(5, 'normal'),
  todo(4, 'high'),
  todo(3, 'normal'),
  todo(2, 'high'),
  todo(1, 'low'),
];
const list = [six, five, four, three, two, one];

function ids(state: TodosState): number[] | undefined {
  return state.todos?.map(({ id }) => id);
}

test.each([
  { id: 2, place: { before: 6 }, want: [2, 6, 5, 4, 3, 1] },
  { id: 6, place: { before: 2 }, want: [5, 4, 3, 6, 2, 1] },
  { id: 6, place: { after: 4 }, want: [5, 4, 6, 3, 2, 1] },
  { id: 1, place: { after: 6 }, want: [6, 1, 5, 4, 3, 2] },
  { id: 3, place: { after: 1 }, want: [6, 5, 4, 2, 1, 3] },
])('puts todo $id at $place in the list, and the rest keep their order', ({ id, place, want }) => {
  const state = new TodosState();
  state.todos = list;

  state.move(id, place);

  expect(ids(state)).toEqual(want);
});

test.each([
  { id: 7, place: { before: 6 } },
  { id: 2, place: { before: 7 } },
  { id: 2, place: { after: 7 } },
])('leaves the list as it is for a todo it lacks: $id at $place', ({ id, place }) => {
  const state = new TodosState();
  state.todos = list;

  state.move(id, place);

  expect(state.todos).toBe(list);
});

test('moves nothing before the list loads', () => {
  const state = new TodosState();

  state.move(2, { before: 6 });

  expect(state.todos).toBeUndefined();
});

/** An editor over the list, open on the todo, with a new title typed and a new priority picked. */
function moving(item: Todo) {
  const state = new TodosState();
  state.todos = list;
  const focusAdd = vi.fn<() => void>();
  const editor = new Editor(state, new Focus(focusAdd));
  editor.open(item);
  editor.draft.title = `${item.title} today`;
  editor.draft.priority = 'low';
  const button = document.body.appendChild(document.createElement('button'));
  return { state, editor, button, focusAdd };
}

function serve(...answers: (Response | Promise<Response>)[]) {
  const fetchMock = vi.fn<typeof fetch>();
  for (const answer of answers) {
    fetchMock.mockReturnValueOnce(Promise.resolve(answer));
  }
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const moved = (item: Todo) => Response.json(item);

test.each([
  { item: four, direction: 'up', body: '{"before":6}', want: [4, 6, 5, 3, 2, 1] },
  { item: four, direction: 'down', body: '{"after":2}', want: [6, 5, 3, 2, 4, 1] },
  { item: three, direction: 'up', body: '{"before":5}', want: [6, 3, 5, 4, 2, 1] },
] as const)(
  'moves todo $item.id $direction in its group, and keeps the draft',
  async ({ item, direction, body, want }) => {
    const fetchMock = serve(moved(item));
    const { state, editor, button } = moving(item);

    await editor.move(direction, button);

    expect(fetchMock).toHaveBeenCalledExactlyOnceWith(`/api/todos/${item.id}`, expect.objectContaining({ body }));
    expect(ids(state)).toEqual(want);
    expect(editor.id).toBe(item.id);
    expect(editor.draft.title).toBe(`${item.title} today`);
    expect(editor.draft.priority).toBe('low');
    expect(editor.status).toEqual({ kind: 'moved', direction });
  }
);

test.each([
  { item: six, direction: 'up' },
  { item: two, direction: 'down' },
  { item: five, direction: 'up' },
  { item: three, direction: 'down' },
  { item: one, direction: 'up' },
  { item: one, direction: 'down' },
] as const)("can't move todo $item.id $direction past its group's end", async ({ item, direction }) => {
  const fetchMock = serve();
  const { state, editor, button } = moving(item);

  expect(editor.canMove(direction)).toBe(false);
  await editor.move(direction, button);

  expect(fetchMock).not.toHaveBeenCalled();
  expect(state.todos).toBe(list);
  expect(editor.status).toEqual({ kind: 'idle' });
});

test('can move a todo either way inside its group, and none with the editor closed', () => {
  const { editor } = moving(four);

  expect(editor.canMove('up')).toBe(true);
  expect(editor.canMove('down')).toBe(true);
  editor.close();
  expect(editor.canMove('up')).toBe(false);
  expect(editor.canMove('down')).toBe(false);
});

test('changes nothing else while a move runs', async () => {
  const answer = Promise.withResolvers<Response>();
  const fetchMock = serve(answer.promise);
  const { state, editor, button } = moving(four);

  const move = editor.move('up', button);
  expect(editor.status).toEqual({ kind: 'moving' });
  expect(editor.busy).toBe(true);
  await editor.move('down', button);
  editor.close();
  editor.open(five);
  await editor.save();
  await editor.delete();
  answer.resolve(moved(four));
  await move;

  expect(fetchMock).toHaveBeenCalledOnce();
  expect(ids(state)).toEqual([4, 6, 5, 3, 2, 1]);
  expect(editor.id).toBe(4);
  expect(editor.busy).toBe(false);
});

test('says the move failed, keeps the list and the draft, and moves on the next try', async () => {
  const fetchMock = serve(new Response('invalid before: no such todo', { status: 400 }), moved(four));
  const { state, editor, button } = moving(four);

  await editor.move('up', button);

  expect(editor.status).toEqual({ kind: 'moveFailed' });
  expect(state.todos).toBe(list);
  expect(editor.id).toBe(4);
  expect(editor.draft.title).toBe('Todo 4 today');

  await editor.move('up', button);

  expect(fetchMock).toHaveBeenCalledTimes(2);
  expect(ids(state)).toEqual([4, 6, 5, 3, 2, 1]);
  expect(editor.status).toEqual({ kind: 'moved', direction: 'up' });
});

test('says the move failed when the gateway can not be reached', async () => {
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));
  const { state, editor, button } = moving(four);

  await editor.move('down', button);

  expect(editor.status).toEqual({ kind: 'moveFailed' });
  expect(state.todos).toBe(list);
});

test('gives the focus back to the button pressed when the move left it on the page', async () => {
  serve(moved(four));
  const { editor, button, focusAdd } = moving(four);

  await editor.move('up', button);

  expect(button).toHaveFocus();
  expect(focusAdd).not.toHaveBeenCalled();
});

test('leaves the focus where the user moved it while the move ran', async () => {
  const answer = Promise.withResolvers<Response>();
  serve(answer.promise);
  const { editor, button } = moving(four);
  const elsewhere = document.body.appendChild(document.createElement('button'));

  const move = editor.move('up', button);
  elsewhere.focus();
  answer.resolve(moved(four));
  await move;
  flushSync();

  expect(elsewhere).toHaveFocus();
});
