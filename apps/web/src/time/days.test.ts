import { afterEach, expect, test, vi } from 'vitest';

import { dayRows, missingNote } from './days';
import type { Day } from './report';

afterEach(() => {
  vi.unstubAllEnvs();
});

// Thursday 15 to Wednesday 21 October 2026, with nothing logged on the Friday or over the weekend.
const days: Day[] = [
  { date: '2026-10-15', loggedHours: 7.5, missing: false },
  { date: '2026-10-16', loggedHours: 0, missing: true },
  { date: '2026-10-17', loggedHours: 0, missing: false },
  { date: '2026-10-18', loggedHours: 0, missing: false },
  { date: '2026-10-19', loggedHours: 8, missing: false },
  { date: '2026-10-20', loggedHours: 6.25, missing: false },
  { date: '2026-10-21', loggedHours: 1.5, missing: false },
];

test("lists the days in the report's order by their dates and weekdays, the last one as today", () => {
  expect(dayRows(days)).toEqual([
    { date: '2026-10-15', label: '15 Oct', title: 'Thursday', hours: '7h 30m', logged: true, missing: false },
    { date: '2026-10-16', label: '16 Oct', title: 'Friday', hours: '0h', logged: false, missing: true },
    { date: '2026-10-17', label: '17 Oct', title: 'Saturday', hours: '0h', logged: false, missing: false },
    { date: '2026-10-18', label: '18 Oct', title: 'Sunday', hours: '0h', logged: false, missing: false },
    { date: '2026-10-19', label: '19 Oct', title: 'Monday', hours: '8h', logged: true, missing: false },
    { date: '2026-10-20', label: '20 Oct', title: 'Tuesday', hours: '6h 15m', logged: true, missing: false },
    { date: '2026-10-21', label: '21 Oct', title: 'Today', hours: '1h 30m', logged: true, missing: false },
  ]);
});

test('names each day as the report does in a time zone behind UTC too', () => {
  vi.stubEnv('TZ', 'America/Los_Angeles');

  expect(dayRows(days.slice(0, 2)).map(({ label, title }) => `${label} ${title}`)).toEqual([
    '15 Oct Thursday',
    '16 Oct Today',
  ]);
});

test('lists no row for no day', () => {
  expect(dayRows([])).toEqual([]);
});

test.each([
  { count: 0, want: undefined },
  { count: 1, want: '1 day missing' },
  { count: 3, want: '3 days missing' },
])('notes $count missing days as "$want"', ({ count, want }) => {
  const marked = days.map((day, index) => ({ ...day, missing: index < count }));

  expect(missingNote(marked)).toBe(want);
});
