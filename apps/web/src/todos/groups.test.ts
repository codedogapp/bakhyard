import { expect, test } from 'vitest';

import type { Priority, Todo } from './api';
import { groupTodos } from './groups';

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
