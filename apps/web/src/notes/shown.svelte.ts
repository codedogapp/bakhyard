import { untrack } from 'svelte';
import type { Attachment } from 'svelte/attachments';

import { fetchNote, saveNote } from './api';
import type { RenderedNote } from './render';
import type { NotesState } from './state.svelte';
import type { Note } from './tree';

/** The note as its page has it: loading, missing from the vault, failed to load, or loaded and rendered. */
type Body =
  | { kind: 'loading' | 'missing' | 'failed' }
  | { kind: 'loaded'; markdown: string; version: string; note: RenderedNote };

/** The editor's last save: running, or how it ended; changed when the note had changed since it loaded. */
export type SaveStatus = 'idle' | 'saving' | 'saved' | 'changed' | 'failed';

/**
 * The note that a note's page shows: the note once it loads, whether the editor shows in its place, and the editor's
 * save (decision 402). The page takes a new one for each note it goes to, so the answer for a note it has left changes
 * nothing.
 */
export class ShownNote {
  body = $state.raw<Body>({ kind: 'loading' });
  /** Whether the note shows in the editor, in place of its rendered text. A note's page opens with the text. */
  editing = $state(false);
  status = $state<SaveStatus>('idle');
  readonly #note: Note;
  readonly #notes: NotesState;
  /** The editor's text since its last change; undefined while it has none. */
  #text: string | undefined;
  /** The Edit button, which turns into Cancel. */
  #button: HTMLButtonElement | undefined;

  /** The note, and the app's state for the Notes pages, whose list the note's links need. */
  constructor(note: Note, notes: NotesState) {
    this.#note = note;
    this.#notes = notes;
  }

  /** Whether a save runs; until it ends, the editor takes no change, and Save and Cancel wait. */
  get busy(): boolean {
    return this.status === 'saving';
  }

  /** An attachment for the Edit button, which turns into Cancel. */
  readonly button: Attachment<HTMLButtonElement> = (button) => {
    this.#button = button;
  };

  /** Loads the note's file, the renderer, a chunk of its own, the first time a note shows, and the vault's list. */
  async load(): Promise<void> {
    this.body = await this.#fetch();
  }

  /**
   * Edit turns into Cancel and back: one button, which keeps the focus. It takes the focus itself, which WebKit doesn't
   * give a button on a click (plan §6.8), so the focus doesn't go with the editor. Edit starts from the note's text.
   */
  toggle(): void {
    if (this.busy) {
      return;
    }
    this.#button?.focus();
    this.#text = undefined;
    this.status = 'idle';
    this.editing = !this.editing;
  }

  /** Takes the editor's text after each change. */
  edit(text: string): void {
    this.#text = text;
  }

  /**
   * Saves the editor's text, if the note hasn't changed since it loaded, then shows the note with it and gives the
   * focus to Edit. A note that has changed, or a failed save, keeps the editor open with the text. With no change to
   * save, it only closes the editor.
   */
  async save(): Promise<void> {
    const body = this.body;
    const text = this.#text;
    if (this.busy || !this.editing || body.kind !== 'loaded') {
      return;
    }
    if (text === undefined || text === body.markdown) {
      this.#close('idle');
      return;
    }
    this.status = 'saving';
    try {
      // Rendered first, so that a save that went through is never said to have failed.
      const { renderNote } = await import('./render');
      const note = renderNote(text, this.#note, this.#notes.files ?? []);
      const version = await saveNote(this.#note.path, text, body.version);
      if (version === undefined) {
        this.status = 'changed';
        return;
      }
      this.body = { kind: 'loaded', markdown: text, version, note };
      this.#close('saved');
    } catch {
      this.status = 'failed';
    }
  }

  #close(status: SaveStatus): void {
    this.#button?.focus();
    this.status = status;
    this.editing = false;
  }

  async #fetch(): Promise<Body> {
    try {
      // Untracked, or the list's arrival would run the page's effect, and load the note, again.
      const listed = untrack(() => this.#notes.load());
      const [file, { renderNote }] = await Promise.all([fetchNote(this.#note.path), import('./render'), listed]);
      if (file === undefined) {
        return { kind: 'missing' };
      }
      const files = this.#notes.files;
      return files === undefined
        ? { kind: 'failed' }
        : { kind: 'loaded', ...file, note: renderNote(file.markdown, this.#note, files) };
    } catch {
      return { kind: 'failed' };
    }
  }
}
