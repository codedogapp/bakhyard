import type { AddStatus } from './state.svelte';

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
