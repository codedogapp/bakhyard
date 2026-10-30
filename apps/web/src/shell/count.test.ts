import { expect, test } from 'vitest';

import { countLabel } from './count';

test.each([
  { count: 0, noun: 'PR', want: undefined },
  { count: 1, noun: 'PR', want: '1 PR' },
  { count: 2, noun: 'PR', want: '2 PRs' },
  { count: 120, noun: 'PR', want: '120 PRs' },
  { count: 0, noun: 'ticket', want: undefined },
  { count: 1, noun: 'ticket', want: '1 ticket' },
  { count: 3, noun: 'ticket', want: '3 tickets' },
])('counts $count of $noun as $want', ({ count, noun, want }) => {
  expect(countLabel(count, noun)).toBe(want);
});
