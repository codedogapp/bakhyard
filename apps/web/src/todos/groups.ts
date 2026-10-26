import type { Priority, Todo } from './api';

/** The priorities in the page's order, which the groups and the editor's options follow. */
export const priorities: readonly Priority[] = ['high', 'normal', 'low'];

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
