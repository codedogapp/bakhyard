import { addTodo, fetchTodos, type Todo } from './api';

/** The last add: running, or how it ended. */
export type AddStatus = { kind: 'idle' | 'adding' | 'added' | 'failed' } | { kind: 'refused'; reason: string };

/** The Todos page's list. The page makes a new one each time it shows, so the list has the todos added since. */
export class TodosState {
  /** The todos, newest first; undefined until the list loads. */
  todos = $state.raw<readonly Todo[]>();
  /** Whether the list failed to load. */
  failed = $state(false);
  /** The new todo's title, as typed in the form. */
  draft = $state('');
  addStatus = $state.raw<AddStatus>({ kind: 'idle' });

  /** Whether an add is running; until it ends, another add does nothing. */
  get busy(): boolean {
    return this.addStatus.kind === 'adding';
  }

  async load(): Promise<void> {
    try {
      this.todos = await fetchTodos();
    } catch {
      this.failed = true;
    }
  }

  /**
   * Adds the draft as a todo, which goes on top of the list, and empties the draft; a refused or failed add keeps it.
   * Adds nothing for a blank draft, or before the list has loaded.
   */
  async add(): Promise<void> {
    if (this.busy || this.todos === undefined || this.draft.trim() === '') {
      return;
    }
    this.addStatus = { kind: 'adding' };
    try {
      const added = await addTodo(this.draft);
      if (added.kind === 'refused') {
        this.addStatus = added;
        return;
      }
      this.todos = [added.todo, ...this.todos];
      this.draft = '';
      this.addStatus = { kind: 'added' };
    } catch {
      this.addStatus = { kind: 'failed' };
    }
  }
}
