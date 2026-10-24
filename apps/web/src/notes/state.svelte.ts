import { SvelteSet } from 'svelte/reactivity';

import { fetchNotes, syncNotes } from './api';

/** The last request for the list: running, or how it ended. A Sync that worked keeps its time. */
export type Status =
  | { kind: 'idle' | 'loading' | 'loaded' | 'load-failed' | 'syncing' | 'sync-failed' }
  | { kind: 'synced'; at: Date };

/**
 * The Notes page's state. App keeps it while it runs, so coming back from a note finds the page as it was left: the
 * same list, filter and open folders.
 */
export class NotesState {
  /** The vault's files from the last list that loaded; undefined until one does. */
  files = $state.raw<readonly string[]>();
  status = $state.raw<Status>({ kind: 'idle' });
  filter = $state('');
  /** The paths of the open folders. */
  readonly open = new SvelteSet<string>();

  /** Whether a request for the list is running; nothing else starts until it ends. */
  get busy(): boolean {
    return this.status.kind === 'loading' || this.status.kind === 'syncing';
  }

  /** Loads the list, unless it has loaded already: after that, only Sync loads it again (decision 71). */
  async load(): Promise<void> {
    if (this.files !== undefined || this.busy) {
      return;
    }
    this.status = { kind: 'loading' };
    try {
      this.files = await fetchNotes();
      this.status = { kind: 'loaded' };
    } catch {
      this.status = { kind: 'load-failed' };
    }
  }

  /** Scans the vault again and lists it as it is now; a failure keeps the list. */
  async sync(): Promise<void> {
    if (this.busy) {
      return;
    }
    this.status = { kind: 'syncing' };
    try {
      this.files = await syncNotes();
      this.status = { kind: 'synced', at: new Date() };
    } catch {
      this.status = { kind: 'sync-failed' };
    }
  }
}
