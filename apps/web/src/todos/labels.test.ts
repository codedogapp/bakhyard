import { expect, test } from 'vitest';

import {
  addLabel,
  editLabel,
  groupNote,
  priorityLabel,
  rowLabel,
  searchedLabel,
  ticketsNote,
  tickLabel,
  todosSummary,
} from './labels';

test.each([
  { count: 0, want: 'No todos on the list.' },
  { count: 1, want: '1 todo on the list.' },
  { count: 2, want: '2 todos on the list.' },
  { count: 120, want: '120 todos on the list.' },
])('sums up $count todos as "$want"', ({ count, want }) => {
  expect(todosSummary(count)).toBe(want);
});

test.each([
  { priority: 'high', want: 'High' },
  { priority: 'normal', want: 'Normal' },
  { priority: 'low', want: 'Low' },
] as const)('names the priority $priority "$want"', ({ priority, want }) => {
  expect(priorityLabel(priority)).toBe(want);
});

test.each([
  { count: 1, want: '1 todo' },
  { count: 2, want: '2 todos' },
  { count: 120, want: '120 todos' },
])('counts a group of $count as "$want"', ({ count, want }) => {
  expect(groupNote(count)).toBe(want);
});

test.each([
  { count: 0, want: undefined },
  { count: 1, want: '1 ticket' },
  { count: 2, want: '2 tickets' },
  { count: 120, want: '120 tickets' },
])('counts $count Jira tickets as $want', ({ count, want }) => {
  expect(ticketsNote(count)).toBe(want);
});

test('tells when a search began in the time zone of the device, Tokyo in the tests', () => {
  expect(searchedLabel('2026-10-26T00:41:00Z')).toBe('26 October at 09:41');
  expect(searchedLabel('2026-12-31T15:05:00Z')).toBe('1 January at 00:05');
});

test('labels no add as blank', () => {
  expect(addLabel(undefined)).toBe('');
});

test.each([
  { status: { kind: 'idle' }, want: '' },
  { status: { kind: 'adding' }, want: 'Adding…' },
  { status: { kind: 'added' }, want: 'Added' },
  {
    status: { kind: 'refused', reason: 'invalid title: over 500 characters' },
    want: 'invalid title: over 500 characters',
  },
  { status: { kind: 'failed' }, want: 'Add failed' },
] as const)('labels the add $status.kind as "$want"', ({ status, want }) => {
  expect(addLabel(status)).toBe(want);
});

test.each([
  { tick: undefined, want: '' },
  { tick: { kind: 'ticking', done: true }, want: '' },
  { tick: { kind: 'ticking', done: false }, want: '' },
  { tick: { kind: 'failed', done: true }, want: 'Tick failed' },
  { tick: { kind: 'failed', done: false }, want: 'Untick failed' },
] as const)('labels the tick $tick as "$want"', ({ tick, want }) => {
  expect(tickLabel(tick)).toBe(want);
});

test.each([
  { tick: undefined, dropFailed: false, want: '' },
  { tick: { kind: 'ticking', done: true }, dropFailed: true, want: 'Move failed' },
  { tick: undefined, dropFailed: true, want: 'Move failed' },
  { tick: { kind: 'failed', done: false }, dropFailed: true, want: 'Untick failed' },
  { tick: { kind: 'failed', done: true }, dropFailed: false, want: 'Tick failed' },
] as const)(
  'labels a row with the tick $tick and a failed drop $dropFailed as "$want"',
  ({ tick, dropFailed, want }) => {
    expect(rowLabel(tick, dropFailed)).toBe(want);
  }
);

test.each([
  { status: { kind: 'idle' }, want: '' },
  { status: { kind: 'saving' }, want: 'Saving…' },
  {
    status: { kind: 'refused', reason: 'invalid title: empty once trimmed' },
    want: 'invalid title: empty once trimmed',
  },
  { status: { kind: 'saveFailed' }, want: 'Save failed' },
  { status: { kind: 'deleting' }, want: 'Deleting…' },
  { status: { kind: 'deleteFailed' }, want: 'Delete failed' },
  { status: { kind: 'moving' }, want: 'Moving…' },
  { status: { kind: 'moved', direction: 'up' }, want: 'Moved up' },
  { status: { kind: 'moved', direction: 'down' }, want: 'Moved down' },
  { status: { kind: 'moveFailed' }, want: 'Move failed' },
] as const)('labels the edit $status.kind as "$want"', ({ status, want }) => {
  expect(editLabel(status)).toBe(want);
});
