import type { Edit, Priority, Todo } from './api';

/** What the row editor holds for its todo until a save: the title as typed, the priority and the ticket as picked. */
export class Draft {
  /** The new title, as typed in the editor. */
  title = $state('');
  /** The new priority, as picked in the editor. */
  priority = $state<Priority>('normal');
  /** The key of the ticket to link, as picked in the editor; empty for none. */
  ticket = $state('');

  /** Starts the draft over from the todo's title, priority and ticket. */
  reset(todo: Todo): void {
    this.title = todo.title;
    this.priority = todo.priority;
    this.ticket = todo.ticket ?? '';
  }

  /**
   * What the draft changes in the todo: its title, its priority, its ticket, which an empty key unlinks, or several;
   * undefined if none changed.
   */
  changes(todo: Todo): Edit | undefined {
    const edit: Edit = {};
    if (this.title !== todo.title) {
      edit.title = this.title;
    }
    if (this.priority !== todo.priority) {
      edit.priority = this.priority;
    }
    if (this.ticket !== (todo.ticket ?? '')) {
      edit.ticket = this.ticket;
    }
    return Object.keys(edit).length > 0 ? edit : undefined;
  }
}
