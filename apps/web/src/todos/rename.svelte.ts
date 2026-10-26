import { flushSync } from 'svelte';
import type { Attachment } from 'svelte/attachments';

import { type Refused, renameTodo, type Todo } from './api';
import type { TodosState } from './state.svelte';

/** The editor's last save: running, or how it ended; a save that succeeds closes the editor. */
export type RenameStatus = { kind: 'idle' | 'saving' | 'failed' } | Refused;

/** Focuses the editor's field as it shows, with the caret after the title. */
export const focusField: Attachment<HTMLInputElement> = (input) => {
  input.focus();
  input.setSelectionRange(input.value.length, input.value.length);
};

/** The Todos page's rename editor, open in one todo's row at a time. */
export class Rename {
  /** The todo whose editor is open; undefined while none is. */
  id = $state<number>();
  /** The new title, as typed in the editor. */
  draft = $state('');
  status = $state.raw<RenameStatus>({ kind: 'idle' });
  readonly #list: TodosState;
  /** The todo whose Edit button takes the focus as it shows again; plain, since nothing on the page shows it. */
  #refocus: number | undefined;

  constructor(list: TodosState) {
    this.#list = list;
  }

  /** Whether a save is running; until it ends, the editor stays open as it is and no other opens. */
  get saving(): boolean {
    return this.status.kind === 'saving';
  }

  /**
   * Opens the todo's editor with its title, and drops any other's draft. The editor shows at once, so its field takes
   * the focus inside the tap's click handler: iOS brings up the keyboard only for such a focus.
   */
  open(todo: Todo): void {
    if (this.saving) {
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
    if (this.saving) {
      return;
    }
    this.#refocus = this.id;
    this.id = undefined;
  }

  /**
   * Saves the draft as the todo's title, which the service trims and checks, and closes the editor; a refused or
   * failed save keeps it open with the draft. A title left as it was closes it with no request; a blank one does nothing.
   */
  async save(): Promise<void> {
    const todo = this.#list.todos?.find(({ id }) => id === this.id);
    if (this.saving || todo === undefined || this.draft.trim() === '') {
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
      this.status = { kind: 'failed' };
    }
  }

  /** Closes the editor on Escape, unless the key ends an IME composition. */
  readonly keydown = (event: KeyboardEvent): void => {
    if (event.key === 'Escape' && !event.isComposing) {
      this.close();
    }
  };

  /** An attachment for the todo's Edit button, which takes the focus as it shows again once the editor has closed. */
  editButton(id: number): Attachment<HTMLButtonElement> {
    return (button) => {
      if (this.#refocus === id) {
        this.#refocus = undefined;
        button.focus();
      }
    };
  }
}
