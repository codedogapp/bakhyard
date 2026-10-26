import type { EditStatus } from './editor.svelte';
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

/** What the last add did: "Adding…", "Added", the service's reason or "Add failed"; blank before the first add. */
export function addLabel(status: AddStatus): string {
  switch (status.kind) {
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

/** What a todo's last tick did, if it failed: "Tick failed" or "Untick failed"; blank otherwise. */
export function tickLabel(tick: Tick | undefined): string {
  if (tick?.kind !== 'failed') {
    return '';
  }
  return tick.done ? 'Tick failed' : 'Untick failed';
}

/** What the editor's last save did, unless it succeeded: "Saving…", the service's reason or "Save failed". */
export function editLabel(status: EditStatus): string {
  switch (status.kind) {
    case 'saving':
      return 'Saving…';
    case 'refused':
      return status.reason;
    case 'failed':
      return 'Save failed';
    default:
      return '';
  }
}
