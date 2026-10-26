import { expect, test } from 'vitest';

import type { Priority, Todo } from './api';
import { groupTodos, stepPlace } from './groups';

function todo(id: number, priority: Priority): Todo {
  return { id, title: `Todo ${id}`, done: false, priority };
}

test('groups the todos by priority, High first, each in the list order', () => {
  const [low, high, normal, older] = [todo(4, 'low'), todo(3, 'high'), todo(2, 'normal'), todo(1, 'high')];

  expect(groupTodos([low, high, normal, older])).toEqual([
    { priority: 'high', todos: [high, older] },
    { priority: 'normal', todos: [normal] },
    { priority: 'low', todos: [low] },
  ]);
});

test('leaves out a priority without todos', () => {
  const [low, high] = [todo(2, 'low'), todo(1, 'high')];

  expect(groupTodos([low, high])).toEqual([
    { priority: 'high', todos: [high] },
    { priority: 'low', todos: [low] },
  ]);
});

test('gives no group without todos', () => {
  expect(groupTodos([])).toEqual([]);
});

// The list: High 6, 4 and 2; Normal 5 and 3; Low 1, alone in its group.
const list = [todo(6, 'high'), todo(5, 'normal'), todo(4, 'high'), todo(3, 'normal'), todo(2, 'high'), todo(1, 'low')];
const groups = groupTodos(list);

test.each([
  { id: 4, direction: 'up', want: { before: 6 } },
  { id: 2, direction: 'up', want: { before: 4 } },
  { id: 3, direction: 'up', want: { before: 5 } },
  { id: 6, direction: 'down', want: { after: 4 } },
  { id: 4, direction: 'down', want: { after: 2 } },
  { id: 5, direction: 'down', want: { after: 3 } },
] as const)('moves todo $id $direction next to the todo it passes in its group', ({ id, direction, want }) => {
  expect(stepPlace(groups, list.find((each) => each.id === id) as Todo, direction)).toEqual(want);
});

test.each([
  { id: 6, direction: 'up' },
  { id: 2, direction: 'down' },
  { id: 5, direction: 'up' },
  { id: 3, direction: 'down' },
  { id: 1, direction: 'up' },
  { id: 1, direction: 'down' },
] as const)("can't move todo $id $direction past its group's end", ({ id, direction }) => {
  expect(stepPlace(groups, list.find((each) => each.id === id) as Todo, direction)).toBeUndefined();
});

test.each(['up', 'down'] as const)("can't move a todo %s that isn't in the groups", (direction) => {
  expect(stepPlace(groups, todo(7, 'normal'), direction)).toBeUndefined();
  expect(stepPlace([], todo(7, 'normal'), direction)).toBeUndefined();
});
