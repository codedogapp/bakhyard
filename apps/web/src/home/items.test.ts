import { describe, expect, test } from 'vitest';

import { countNote, dueLabel, itemHref } from './items';

describe('dueLabel', () => {
  const now = new Date('2026-10-20T01:00:00Z');
  const second = 1000;
  const minute = 60 * second;
  const hour = 60 * minute;
  const day = 24 * hour;

  test.each([
    { offset: 0, want: 'now' },
    { offset: 30 * second - 1, want: 'now' },
    { offset: -(30 * second - 1), want: 'now' },
    { offset: 30 * second, want: 'in 1 minute' },
    { offset: -30 * second, want: '1 minute ago' },
    { offset: 59 * minute + 30 * second - 1, want: 'in 59 minutes' },
    { offset: -(59 * minute + 30 * second - 1), want: '59 minutes ago' },
    { offset: 59 * minute + 30 * second, want: 'in 1 hour' },
    { offset: -(59 * minute + 30 * second), want: '1 hour ago' },
    { offset: 3 * hour, want: 'in 3 hours' },
    { offset: -2 * hour, want: '2 hours ago' },
    { offset: 23 * hour + 30 * minute - 1, want: 'in 23 hours' },
    { offset: -(23 * hour + 30 * minute - 1), want: '23 hours ago' },
    { offset: 23 * hour + 30 * minute, want: 'in 1 day' },
    { offset: -(23 * hour + 30 * minute), want: '1 day ago' },
    { offset: 5 * day, want: 'in 5 days' },
    { offset: -40 * day, want: '40 days ago' },
  ])('says "$want" for a due time $offset ms from now', ({ offset, want }) => {
    const due = new Date(now.getTime() + offset).toISOString();

    expect(dueLabel(due, now)).toBe(want);
  });

  test('reads a due time with an offset', () => {
    expect(dueLabel('2026-10-20T13:00:00+09:00', now)).toBe('in 3 hours');
  });
});

describe('countNote', () => {
  test.each([
    { count: 6, want: '6 items · most urgent first' },
    { count: 1, want: '1 item · most urgent first' },
    { count: 0, want: undefined },
  ])('says "$want" for $count items', ({ count, want }) => {
    expect(countNote(count)).toBe(want);
  });
});

describe('itemHref', () => {
  test("links to the item's app's page", () => {
    expect(itemHref({ app: 'pulls' })).toBe('/pulls');
  });
});
