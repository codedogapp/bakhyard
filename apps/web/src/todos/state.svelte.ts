import { SvelteMap } from 'svelte/reactivity';

import { addTodo, fetchTodos, type Refused, type Todo, tickTodo } from './api';

/** The last add: running, or how it ended. */
export type AddStatus = { kind: 'idle' | 'adding' | 'added' | 'failed' } | Refused;

/** A todo's last tick, to done or not, while it runs or once it has failed; a failed tick put the done mark back. */
export interface Tick {
  kind: 'ticking' | 'failed';
  done: boolean;
}

/** The Todos page's list. The page makes a new one each time it shows, so the list has the todos added since. */
export class TodosState {
  /** The todos, newest first; undefined until the list loads. */
  todos = $state.raw<readonly Todo[]>();
  /** Whether the list failed to load. */
  failed = $state(false);
  /** The new todo's title, as typed in the form. */
  draft = $state('');
  addStatus = $state.raw<AddStatus>({ kind: 'idle' });
  /** The ticks that run or have failed, by the todo's ID; a tick that succeeds leaves none. */
  readonly ticks = new SvelteMap<number, Tick>();

  /** Whether an add is running; until it ends, another add does nothing. */
  get busy(): boolean {
    return this.addStatus.kind === 'adding';
  }

  /** Whether a tick of the todo is running; until it ends, another tick of it does nothing. */
  ticking(id: number): boolean {
    return this.ticks.get(id)?.kind === 'ticking';
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

  /**
   * Ticks the todo done, or unticks it, at once, then saves it; a failed save puts the done mark back. The answer
   * changes only the done mark, so it can't undo a rename whose answer came first.
   */
  async tick(id: number, done: boolean): Promise<void> {
    if (this.ticking(id)) {
      return;
    }
    this.ticks.set(id, { kind: 'ticking', done });
    this.change(id, (todo) => ({ ...todo, done }));
    try {
      const saved = await tickTodo(id, done);
      this.change(id, (todo) => ({ ...todo, done: saved.done }));
      this.ticks.delete(id);
    } catch {
      this.change(id, (todo) => ({ ...todo, done: !done }));
      this.ticks.set(id, { kind: 'failed', done });
    }
  }

  /** Puts the renamed todo's title in the list; only the title, so it can't undo a tick whose answer came first. */
  retitle(renamed: Todo): void {
    this.change(renamed.id, (todo) => ({ ...todo, title: renamed.title }));
  }

  /** Takes the deleted todo off the list. */
  remove(id: number): void {
    this.todos = this.todos?.filter((todo) => todo.id !== id);
  }

  private change(id: number, change: (todo: Todo) => Todo): void {
    this.todos = this.todos?.map((todo) => (todo.id === id ? change(todo) : todo));
  }
}
