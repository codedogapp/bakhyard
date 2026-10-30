import { expect, test } from 'vitest';

import { dayLabel } from './day';

// The tests run in Tokyo, 9 hours ahead of UTC, where 31 October begins at 2026-10-30T15:00:00Z.
test.each([
  { time: '2026-10-30T15:00:00Z', want: 'Today' },
  { time: '2026-10-31T00:40:00Z', want: 'Today' },
  { time: '2026-10-30T14:59:59Z', want: 'Yesterday' },
  { time: '2026-10-29T15:00:00Z', want: 'Yesterday' },
  { time: '2026-10-29T14:59:59Z', want: '29 Oct' },
  { time: '2026-10-17T03:00:00Z', want: '17 Oct' },
  { time: '2026-11-01T01:00:00Z', want: '1 Nov' },
])('labels $time "$want"', ({ time, want }) => {
  expect(dayLabel(time, new Date('2026-10-31T09:41:00+09:00'))).toBe(want);
});

test('names the day across the turn of a month and a year', () => {
  expect(dayLabel('2026-12-31T03:00:00Z', new Date('2027-01-01T09:00:00+09:00'))).toBe('Yesterday');
  expect(dayLabel('2026-12-30T03:00:00Z', new Date('2027-01-01T09:00:00+09:00'))).toBe('30 Dec');
  // At 05:00 on New Year's Day in Tokyo, while UTC is still on the old year's last day.
  expect(dayLabel('2026-12-31T20:00:00Z', new Date('2027-01-01T10:00:00+09:00'))).toBe('Today');
});
