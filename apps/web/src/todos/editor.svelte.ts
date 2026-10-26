import { flushSync } from 'svelte';
import type { Attachment } from 'svelte/attachments';
import { on } from 'svelte/events';

import { deleteTodo, type Refused, renameTodo, type Todo } from './api';
import type { Focus } from './focus';
import type { TodosState } from './state.svelte';

/** The editor's last save or delete: running, or how it ended; one that succeeds closes the editor. */
export type EditStatus = { kind: 'idle' | 'saving' | 'saveFailed' | 'deleting' | 'deleteFailed' } | Refused;

/** The Todos page's editor, open in one todo's row at a time, which renames or deletes the todo. */
export class Editor {
  /** The todo whose editor is open; undefined while none is. */
  id = $state<number>();
  /** The new title, as typed in the editor. */
  draft = $state('');
  status = $state.raw<EditStatus>({ kind: 'idle' });
  readonly #list: TodosState;
  readonly #focus: Focus;
  /** The page's dialog, which asks before a delete. */
  #dialog: HTMLDialogElement | undefined;
  /** The open editor's Delete button, which opens the dialog. */
  #deleteButton: HTMLButtonElement | undefined;

  constructor(list: TodosState, focus: Focus) {
    this.#list = list;
    this.#focus = focus;
  }

  /** The todo whose editor is open, as the list has it. */
  get todo(): Todo | undefined {
    return this.#list.todos?.find(({ id }) => id === this.id);
  }

  /** Whether a save or a delete is running; until it ends, the editor stays open as it is and no other opens. */
  get busy(): boolean {
    return this.status.kind === 'saving' || this.status.kind === 'deleting';
  }

  /**
   * Opens the todo's editor with its title, and drops any other's draft. The editor shows at once, so its field takes
   * the focus inside the tap's click handler: iOS brings up the keyboard only for such a focus.
   */
  open(todo: Todo): void {
    if (this.busy) {
      return;
    }
    flushSync(() => {
      this.id = todo.id;
      this.draft = todo.title;
      this.status = { kind: 'idle' };
    });
  }

  /** Closes the editor without saving, and gives the focus back to its row's Edit button. */
  close(): void {
    const id = this.id;
    if (this.busy || id === undefined) {
      return;
    }
    flushSync(() => {
      this.id = undefined;
    });
    this.#focus.recoverEdit(id);
  }

  /**
   * Saves the draft as the todo's title, which the service trims and checks, and closes the editor; a refused or
   * failed save keeps it open with the draft. A title left as it was closes it with no request; a blank one does nothing.
   */
  async save(): Promise<void> {
    const todo = this.todo;
    if (this.busy || todo === undefined || this.draft.trim() === '') {
      return;
    }
    if (this.draft === todo.title) {
      this.close();
      return;
    }
    this.status = { kind: 'saving' };
    try {
      const renamed = await renameTodo(todo.id, this.draft);
      if (renamed.kind === 'refused') {
        this.status = renamed;
        return;
      }
      this.#list.retitle(renamed.todo);
      this.status = { kind: 'idle' };
      this.close();
    } catch {
      this.status = { kind: 'saveFailed' };
    }
  }

  /** Asks in the page's dialog whether to delete the todo. */
  confirmDelete(): void {
    if (!this.busy) {
      this.#dialog?.showModal();
    }
  }

  /** Closes the dialog without deleting the todo. */
  cancelDelete(): void {
    this.#dialog?.close();
  }

  /**
   * Closes the dialog and deletes the todo for good, then takes it off the list, closes the editor and gives the focus
   * to the next row's Edit button, else the previous row's, else the Add field. A failed delete keeps the editor open.
   */
  async delete(): Promise<void> {
    const id = this.id;
    this.#dialog?.close();
    if (this.busy || id === undefined) {
      return;
    }
    this.status = { kind: 'deleting' };
    try {
      await deleteTodo(id);
    } catch {
      this.status = { kind: 'deleteFailed' };
      return;
    }
    // Found once the delete has landed, since a todo may have been added meanwhile.
    const neighbour = this.#neighbour(id);
    flushSync(() => {
      this.#list.remove(id);
      this.id = undefined;
      this.status = { kind: 'idle' };
    });
    this.#focus.recoverEdit(neighbour);
  }

  /** Closes the editor on Escape, unless the key ends an IME composition. */
  readonly keydown = (event: KeyboardEvent): void => {
    if (event.key === 'Escape' && !event.isComposing) {
      this.close();
    }
  };

  /**
   * An attachment for the page's dialog, which asks before a delete. As the dialog closes, the focus goes back to the
   * editor's Delete button where the browser didn't put it back: WebKit doesn't after a click, which focuses no button.
   */
  readonly dialog: Attachment<HTMLDialogElement> = (dialog) => {
    this.#dialog = dialog;
    return on(dialog, 'close', () => this.#focus.recover(this.#deleteButton));
  };

  /** An attachment for the open editor's Delete button. */
  readonly deleteButton: Attachment<HTMLButtonElement> = (button) => {
    this.#deleteButton = button;
  };

  /** The ID of the todo after this one on the page, else of the one before it, across the groups. */
  #neighbour(id: number): number | undefined {
    const todos = this.#list.groups.flatMap((group) => group.todos);
    const at = todos.findIndex((todo) => todo.id === id);
    return (todos[at + 1] ?? todos[at - 1])?.id;
  }
}
