import type { Edit, Priority, Todo } from './api';

/** What the row editor holds for its todo until a save: the title as typed and the priority as picked. */
export class Draft {
  /** The new title, as typed in the editor. */
  title = $state('');
  /** The new priority, as picked in the editor. */
  priority = $state<Priority>('normal');

  /** Starts the draft over from the todo's title and priority. */
  reset(todo: Todo): void {
    this.title = todo.title;
    this.priority = todo.priority;
  }

  /** What the draft changes in the todo, its title, its priority or both; undefined if neither changed. */
  changes(todo: Todo): Edit | undefined {
    const edit: Edit = {};
    if (this.title !== todo.title) {
      edit.title = this.title;
    }
    if (this.priority !== todo.priority) {
      edit.priority = this.priority;
    }
    return Object.keys(edit).length > 0 ? edit : undefined;
  }
}
