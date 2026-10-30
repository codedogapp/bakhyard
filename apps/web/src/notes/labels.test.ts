import { expect, test } from 'vitest';

import { countLabel, notesSummary, saveLabel, syncLabel } from './labels';
import type { SaveStatus } from './shown.svelte';
import type { Status } from './state.svelte';

test.each([
  { count: 0, want: 'No notes in the vault.' },
  { count: 1, want: '1 note in the vault.' },
  { count: 236, want: '236 notes in the vault.' },
])('sums up $count notes as "$want"', ({ count, want }) => {
  expect(notesSummary(count)).toBe(want);
});

test.each([
  { count: 1, want: '1 note' },
  { count: 12, want: '12 notes' },
])('counts $count as "$want"', ({ count, want }) => {
  expect(countLabel(count)).toBe(want);
});

test.each<{ status: Status; want: string }>([
  { status: { kind: 'idle' }, want: '' },
  { status: { kind: 'loading' }, want: '' },
  { status: { kind: 'loaded' }, want: '' },
  { status: { kind: 'load-failed' }, want: '' },
  { status: { kind: 'syncing' }, want: 'Syncing…' },
  // 14:02 in Tokyo, the tests' time zone.
  { status: { kind: 'synced', at: new Date('2026-10-24T05:02:59Z') }, want: 'Synced at 14:02' },
  { status: { kind: 'sync-failed' }, want: 'Sync failed' },
])('labels $status.kind as "$want"', ({ status, want }) => {
  expect(syncLabel(status)).toBe(want);
});

test.each<{ status: SaveStatus; want: string }>([
  { status: 'idle', want: '' },
  { status: 'saving', want: 'Saving…' },
  { status: 'saved', want: 'Saved' },
  { status: 'changed', want: 'Not saved: the note has changed since it loaded' },
  { status: 'failed', want: 'Save failed' },
])('labels the save $status as "$want"', ({ status, want }) => {
  expect(saveLabel(status)).toBe(want);
});
