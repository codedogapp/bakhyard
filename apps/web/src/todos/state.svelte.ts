import { fetchTodos, type Todo } from './api';

/** The Todos page's list. The page makes a new one each time it shows, so the list has the todos added since. */
export class TodosState {
  /** The todos, newest first; undefined until the list loads. */
  todos = $state.raw<readonly Todo[]>();
  /** Whether the list failed to load. */
  failed = $state(false);

  async load(): Promise<void> {
    try {
      this.todos = await fetchTodos();
    } catch {
      this.failed = true;
    }
  }
}
