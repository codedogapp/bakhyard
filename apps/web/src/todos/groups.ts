import type { Place, Priority, Todo } from './api';

/** The priorities in the page's order, which the groups and the editor's options follow. */
export const priorities: readonly Priority[] = ['high', 'normal', 'low'];

/** Which way the editor moves a todo in its group: up, towards the top of the page, or down. */
export type Direction = 'up' | 'down';

/** A priority's todos, in the list's order. */
export interface Group {
  priority: Priority;
  todos: readonly Todo[];
}

/** The todos in a group per priority, High first, each in the list's order; a priority without todos has no group. */
export function groupTodos(todos: readonly Todo[]): Group[] {
  return priorities
    .map((priority) => ({ priority, todos: todos.filter((todo) => todo.priority === priority) }))
    .filter((group) => group.todos.length > 0);
}

/**
 * Where a move one step up or down its group puts the todo: right before the todo above it, or right after the one
 * below it, so the rest of the list keeps its order; undefined at the group's end.
 */
export function stepPlace(groups: readonly Group[], todo: Todo, direction: Direction): Place | undefined {
  const todos = groups.find(({ priority }) => priority === todo.priority)?.todos ?? [];
  const at = todos.findIndex(({ id }) => id === todo.id);
  const other = todos[direction === 'up' ? at - 1 : at + 1];
  if (at < 0 || other === undefined) {
    return undefined;
  }
  return direction === 'up' ? { before: other.id } : { after: other.id };
}
