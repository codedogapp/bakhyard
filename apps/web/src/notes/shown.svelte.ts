import { untrack } from 'svelte';
import type { Attachment } from 'svelte/attachments';

import { createNote, fetchNote, historyUrl, saveNote } from './api';
import type { RenderedNote } from './render';
import type { NotesState } from './state.svelte';
import type { Note } from './tree';

/** The note as its page has it: loading, missing from the vault, failed to load, or loaded and rendered. */
type Body =
  | { kind: 'loading' | 'missing' | 'failed' }
  | { kind: 'loaded'; markdown: string; version: string; note: RenderedNote };

/**
 * The editor's last save: running, or how it ended; changed when the note had changed since it loaded, and kept when
 * the editor's text went to a new note instead, at path.
 */
export type SaveStatus = { kind: 'idle' | 'saving' | 'saved' | 'changed' | 'failed' } | { kind: 'kept'; path: string };

/**
 * The note that a note's page shows: the note once it loads, whether the editor shows in its place, and the editor's
 * save (decision 402). The page takes a new one for each note it goes to, so the answer for a note it has left changes
 * nothing.
 */
export class ShownNote {
  body = $state.raw<Body>({ kind: 'loading' });
  /** Whether the note shows in the editor, in place of its rendered text. A note's page opens with the text. */
  editing = $state(false);
  status = $state.raw<SaveStatus>({ kind: 'idle' });
  /**
   * Whether a save found that the note has changed since it loaded, so the editor offers to keep its text as a new note
   * until it closes (decision 424).
   */
  stale = $state(false);
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

  /** Whether a save, or a new note, runs; until it ends, the editor takes no change, and its buttons wait. */
  get busy(): boolean {
    return this.status.kind === 'saving';
  }

  /**
   * Whether the page has the editor's buttons: while notes saves notes, and while the editor is open, even if a sync
   * says that notes no longer saves.
   */
  get editable(): boolean {
    return this.#notes.editable || this.editing;
  }

  /**
   * The web address of the note's history in Gitea (decision 425); undefined when notes has none, and while the editor
   * is open, since leaving the page would drop its text.
   */
  get history(): string | undefined {
    const history = this.#notes.history;
    return history === undefined || this.editing ? undefined : historyUrl(history, this.#note.path);
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
    this.status = { kind: 'idle' };
    this.stale = false;
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
      this.#close({ kind: 'idle' });
      return;
    }
    this.status = { kind: 'saving' };
    try {
      // Rendered first, so that a save that went through is never said to have failed.
      const { renderNote } = await import('./render');
      const note = renderNote(text, this.#note, this.#notes.files ?? []);
      const version = await saveNote(this.#note.path, text, body.version);
      if (version === undefined) {
        this.stale = true;
        this.status = { kind: 'changed' };
        return;
      }
      this.body = { kind: 'loaded', markdown: text, version, note };
      this.#close({ kind: 'saved' });
    } catch {
      this.status = { kind: 'failed' };
    }
  }

  /**
   * Keeps the editor's text as a new note beside the one that changed since it loaded, at the note's name or, as that's
   * taken, the first free one after it (decision 424). Then it shows the note as it is now, or as it was if it fails to
   * load again, and gives the focus to Edit. A failure keeps the editor open with the text and the offer.
   */
  async keep(): Promise<void> {
    const text = this.#text;
    if (this.busy || !this.stale || text === undefined) {
      return;
    }
    this.status = { kind: 'saving' };
    try {
      const path = await createNote(this.#note.path, text);
      const body = await this.#fetch();
      if (body.kind === 'loaded') {
        this.body = body;
      }
      this.#close({ kind: 'kept', path });
    } catch {
      this.status = { kind: 'failed' };
    }
  }

  #close(status: SaveStatus): void {
    this.#button?.focus();
    this.status = status;
    this.stale = false;
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
