import { expect, test } from 'vitest';

import { addLabel, todosSummary } from './labels';

test.each([
  { count: 0, want: 'No todos on the list.' },
  { count: 1, want: '1 todo on the list.' },
  { count: 2, want: '2 todos on the list.' },
  { count: 120, want: '120 todos on the list.' },
])('sums up $count todos as "$want"', ({ count, want }) => {
  expect(todosSummary(count)).toBe(want);
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
