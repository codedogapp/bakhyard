import type { Attachment } from 'svelte/attachments';

/** Focuses the editor's field as it shows, with the caret after the title. */
export const focusField: Attachment<HTMLInputElement> = (input) => {
  input.focus();
  input.setSelectionRange(input.value.length, input.value.length);
};

/** Where the Todos page gives the focus back once a row's editor closes or its todo goes. */
export class Focus {
  /** Gives the focus to the Add field, once a delete has left no row to give it to. */
  readonly #focusAdd: () => void;
  /** The rows' Edit buttons, by their todo's ID; plain, since nothing on the page shows it. */
  readonly #editButtons = new Map<number, HTMLButtonElement>();

  constructor(focusAdd: () => void) {
    this.#focusAdd = focusAdd;
  }

  /** An attachment for the todo's Edit button, which can then take the focus back. */
  editButton(id: number): Attachment<HTMLButtonElement> {
    return (button) => {
      this.#editButtons.set(id, button);
      return () => {
        this.#editButtons.delete(id);
      };
    };
  }

  /** Gives the focus to the todo's Edit button, or to the Add field without one, if the focus was lost. */
  recoverEdit(id: number | undefined): void {
    this.recover(id === undefined ? undefined : this.#editButtons.get(id));
  }

  /**
   * Gives the focus to the button, or to the Add field without one, if the focus went with the closed editor or stayed
   * in a closed dialog, as WebKit leaves it after a click: the user may have moved on while a save or a delete ran.
   */
  recover(button: HTMLButtonElement | undefined): void {
    const focused = document.activeElement;
    if (focused !== document.body && !focused?.closest('dialog:not([open])')) {
      return;
    }
    if (button) {
      button.focus();
    } else {
      this.#focusAdd();
    }
  }
}
