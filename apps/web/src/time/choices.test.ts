import { expect, test } from 'vitest';

import type { Ticket } from '../jira/tickets';
import { dayChoices, defaultDay, hourChoices, minutesLabel, ticketChoices, ticketLabel, workday } from './choices';
import type { Day } from './report';

// Thursday 15 to Wednesday 21 October 2026, with nothing logged on the Friday or the Monday.
const days: Day[] = [
  { date: '2026-10-15', loggedHours: 7.5, missing: false },
  { date: '2026-10-16', loggedHours: 0, missing: true },
  { date: '2026-10-17', loggedHours: 0, missing: false },
  { date: '2026-10-18', loggedHours: 0, missing: false },
  { date: '2026-10-19', loggedHours: 0, missing: true },
  { date: '2026-10-20', loggedHours: 6.25, missing: false },
  { date: '2026-10-21', loggedHours: 0, missing: false },
];

test("offers the report's days in its order, the last one as today", () => {
  expect(dayChoices(days)).toEqual([
    { value: '2026-10-15', label: 'Thursday 15 Oct' },
    { value: '2026-10-16', label: 'Friday 16 Oct' },
    { value: '2026-10-17', label: 'Saturday 17 Oct' },
    { value: '2026-10-18', label: 'Sunday 18 Oct' },
    { value: '2026-10-19', label: 'Monday 19 Oct' },
    { value: '2026-10-20', label: 'Tuesday 20 Oct' },
    { value: '2026-10-21', label: 'Today' },
  ]);
});

test('offers no day for no day', () => {
  expect(dayChoices([])).toEqual([]);
});

test('starts on the earliest day that misses its hours', () => {
  expect(defaultDay(days)).toBe(days[1]);
});

test('starts on today when no day misses its hours', () => {
  const logged = days.map((day) => ({ ...day, missing: false }));

  expect(defaultDay(logged)).toBe(logged[6]);
});

test('starts on no day without days', () => {
  expect(defaultDay([])).toBeUndefined();
});

test('offers each quarter of an hour from 0h 15m to 12h', () => {
  expect(hourChoices).toHaveLength(48);
  expect(hourChoices.slice(0, 5)).toEqual([
    { value: 15, label: '0h 15m' },
    { value: 30, label: '0h 30m' },
    { value: 45, label: '0h 45m' },
    { value: 60, label: '1h' },
    { value: 75, label: '1h 15m' },
  ]);
  expect(hourChoices.at(-1)).toEqual({ value: 720, label: '12h' });
});

test("starts on a working day's 8 hours, which the hours offer", () => {
  expect(workday).toBe(480);
  expect(hourChoices).toContainEqual({ value: workday, label: '8h' });
});

test.each([
  { minutes: 15, want: '0h 15m' },
  { minutes: 480, want: '8h' },
  { minutes: 510, want: '8h 30m' },
])('names $minutes minutes "$want"', ({ minutes, want }) => {
  expect(minutesLabel(minutes)).toBe(want);
});

const ticket = (key: string, summary: string): Ticket => ({
  key,
  summary,
  status: 'In Progress',
  category: 'in-progress',
  url: `https://example.atlassian.net/browse/${key}`,
});

test('names a ticket by its key and summary', () => {
  expect(ticketLabel(ticket('WEB-9', 'Fix the date picker in Safari'))).toBe('WEB-9 · Fix the date picker in Safari');
});

test('offers the tickets by key, in their order', () => {
  const tickets = [ticket('WEB-12', 'Show the order history'), ticket('OPS-31', 'Rotate the certificates')];

  expect(ticketChoices(tickets)).toEqual([
    { value: 'WEB-12', label: 'WEB-12 · Show the order history' },
    { value: 'OPS-31', label: 'OPS-31 · Rotate the certificates' },
  ]);
  expect(ticketChoices([])).toEqual([]);
});
