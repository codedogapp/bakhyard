import { expect, test } from 'vitest';

import { addLabel, editLabel, groupNote, priorityLabel, tickLabel, todosSummary } from './labels';

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
  { status: { kind: 'idle' }, want: '' },
  { status: { kind: 'saving' }, want: 'Saving…' },
  {
    status: { kind: 'refused', reason: 'invalid title: empty once trimmed' },
    want: 'invalid title: empty once trimmed',
  },
  { status: { kind: 'saveFailed' }, want: 'Save failed' },
  { status: { kind: 'deleting' }, want: 'Deleting…' },
  { status: { kind: 'deleteFailed' }, want: 'Delete failed' },
] as const)('labels the edit $status.kind as "$want"', ({ status, want }) => {
  expect(editLabel(status)).toBe(want);
});
