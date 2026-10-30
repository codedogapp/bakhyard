import type { Priority } from './api';
import type { EditStatus } from './editor.svelte';
import type { InboxStatus } from './inbox.svelte';
import type { AddStatus, Tick } from './state.svelte';

/** The Todos page's summary: "3 todos on the list.", "1 todo on the list." or "No todos on the list." */
export function todosSummary(count: number): string {
  switch (count) {
    case 0:
      return 'No todos on the list.';
    case 1:
      return '1 todo on the list.';
    default:
      return `${count} todos on the list.`;
  }
}

const priorityLabels: Record<Priority, string> = { high: 'High', normal: 'Normal', low: 'Low' };

/** A priority's name: "High", "Normal" or "Low". */
export function priorityLabel(priority: Priority): string {
  return priorityLabels[priority];
}

/** A priority group's count: "1 todo" or "3 todos". */
export function groupNote(count: number): string {
  return count === 1 ? '1 todo' : `${count} todos`;
}

/**
 * What the last add did: "Adding…", "Added", the service's reason or "Add failed"; blank before the first add, as for
 * no status.
 */
export function addLabel(status: AddStatus | undefined): string {
  switch (status?.kind) {
    case 'adding':
      return 'Adding…';
    case 'added':
      return 'Added';
    case 'refused':
      return status.reason;
    case 'failed':
      return 'Add failed';
    default:
      return '';
  }
}

/** What an inbox row's last add or dismiss did: as `addLabel` says for an add, else "Dismissing…" or "Dismiss failed". */
export function inboxLabel(status: InboxStatus | undefined): string {
  switch (status?.kind) {
    case 'dismissing':
      return 'Dismissing…';
    case 'dismissFailed':
      return 'Dismiss failed';
    default:
      return addLabel(status);
  }
}

/** What a todo's last tick did, if it failed: "Tick failed" or "Untick failed"; blank otherwise. */
export function tickLabel(tick: Tick | undefined): string {
  if (tick?.kind !== 'failed') {
    return '';
  }
  return tick.done ? 'Tick failed' : 'Untick failed';
}

/** What a row's status line says: a failed tick's text, else "Move failed" if the todo's last drop failed. */
export function rowLabel(tick: Tick | undefined, dropFailed: boolean): string {
  return tickLabel(tick) || (dropFailed ? 'Move failed' : '');
}

/**
 * What the editor's last save, delete or move did, unless a save or a delete succeeded: "Saving…", the service's
 * reason, "Save failed", "Deleting…", "Delete failed", "Moving…", "Moved up", "Moved down" or "Move failed".
 */
export function editLabel(status: EditStatus): string {
  switch (status.kind) {
    case 'saving':
      return 'Saving…';
    case 'refused':
      return status.reason;
    case 'saveFailed':
      return 'Save failed';
    case 'deleting':
      return 'Deleting…';
    case 'deleteFailed':
      return 'Delete failed';
    case 'moving':
      return 'Moving…';
    case 'moved':
      return status.direction === 'up' ? 'Moved up' : 'Moved down';
    case 'moveFailed':
      return 'Move failed';
    default:
      return '';
  }
}
