import { timeLabel } from '../shell/clock';
import type { Status } from './state.svelte';

/** A number of notes: "1 note", "12 notes". */
export function countLabel(count: number): string {
  return count === 1 ? '1 note' : `${count} notes`;
}

/** The Notes page's summary: "236 notes in the vault.", or "No notes in the vault." */
export function notesSummary(count: number): string {
  return `${count === 0 ? 'No notes' : countLabel(count)} in the vault.`;
}

/** What Sync is doing, or did last: "Syncing…", "Synced at 14:02" or "Sync failed"; blank before the first Sync. */
export function syncLabel(status: Status): string {
  switch (status.kind) {
    case 'syncing':
      return 'Syncing…';
    case 'synced':
      return `Synced at ${timeLabel(status.at)}`;
    case 'sync-failed':
      return 'Sync failed';
    default:
      return '';
  }
}
