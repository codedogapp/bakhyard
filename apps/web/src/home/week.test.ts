import { expect, test } from 'vitest';

import { hoursLabel, weekPercent } from './week';

test.each([
  { logged: 31.5, target: 40, want: '31h 30m / 40h' },
  { logged: 32, target: 37.5, want: '32h / 37h 30m' },
  { logged: 0.25, target: 40, want: '0h 15m / 40h' },
  { logged: 0, target: 40, want: '0h / 40h' },
  // 7 h 20 min, which isn't exact in binary.
  { logged: 7 + 1 / 3, target: 40, want: '7h 20m / 40h' },
  // 29.4 and 29.6 seconds short of 8 h.
  { logged: 8 - 29.4 / 3600, target: 40, want: '8h / 40h' },
  { logged: 8 - 30.6 / 3600, target: 40, want: '7h 59m / 40h' },
  { logged: 45, target: 40, want: '45h / 40h' },
])('reads $logged of $target hours as "$want"', ({ logged, target, want }) => {
  expect(hoursLabel({ loggedHours: logged, targetHours: target })).toBe(want);
});

test.each([
  { logged: 30, target: 40, want: 75 },
  { logged: 0, target: 40, want: 0 },
  { logged: 40, target: 40, want: 100 },
  { logged: 45, target: 40, want: 100 },
  { logged: 8, target: 0, want: 0 },
])('fills $want% of the bar for $logged of $target hours', ({ logged, target, want }) => {
  expect(weekPercent({ loggedHours: logged, targetHours: target })).toBe(want);
});
