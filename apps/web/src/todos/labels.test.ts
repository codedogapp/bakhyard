import { expect, test } from 'vitest';

import { todosSummary } from './labels';

test.each([
  { count: 0, want: 'No todos on the list.' },
  { count: 1, want: '1 todo on the list.' },
  { count: 2, want: '2 todos on the list.' },
  { count: 120, want: '120 todos on the list.' },
])('sums up $count todos as "$want"', ({ count, want }) => {
  expect(todosSummary(count)).toBe(want);
});
