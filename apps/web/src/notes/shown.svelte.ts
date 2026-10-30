import { untrack } from 'svelte';

import { fetchNote } from './api';
import type { RenderedNote } from './render';
import type { NotesState } from './state.svelte';
import type { Note } from './tree';

/** The note as its page has it: loading, missing from the vault, failed to load, or loaded and rendered. */
type Body = { kind: 'loading' | 'missing' | 'failed' } | { kind: 'loaded'; markdown: string; note: RenderedNote };

/**
 * The note that a note's page shows: the note once it loads, and whether the editor shows in its place. The page takes
 * a new one for each note it goes to, so the answer for a note it has left changes nothing.
 */
export class ShownNote {
  body = $state.raw<Body>({ kind: 'loading' });
  /** Whether the note shows in the editor, in place of its rendered text. A note's page opens with the text. */
  editing = $state(false);
  readonly #note: Note;
  readonly #notes: NotesState;

  /** The note, and the app's state for the Notes pages, whose list the note's links need. */
  constructor(note: Note, notes: NotesState) {
    this.#note = note;
    this.#notes = notes;
  }

  /** Loads the note's file, the renderer, a chunk of its own, the first time a note shows, and the vault's list. */
  async load(): Promise<void> {
    this.body = await this.#fetch();
  }

  /**
   * Edit turns into Cancel and back: one button, which keeps the focus. It takes the focus itself, which WebKit doesn't
   * give a button on a click (plan §6.8), so the focus doesn't go with the editor.
   */
  toggle(button: HTMLButtonElement): void {
    button.focus();
    this.editing = !this.editing;
  }

  async #fetch(): Promise<Body> {
    try {
      // Untracked, or the list's arrival would run the page's effect, and load the note, again.
      const listed = untrack(() => this.#notes.load());
      const [markdown, { renderNote }] = await Promise.all([fetchNote(this.#note.path), import('./render'), listed]);
      if (markdown === undefined) {
        return { kind: 'missing' };
      }
      const files = this.#notes.files;
      return files === undefined
        ? { kind: 'failed' }
        : { kind: 'loaded', markdown, note: renderNote(markdown, this.#note, files) };
    } catch {
      return { kind: 'failed' };
    }
  }
}
