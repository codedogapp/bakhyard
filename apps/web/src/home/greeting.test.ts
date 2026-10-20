import { describe, expect, test } from 'vitest';

import { dateLabel, greeting, summary } from './greeting';

describe('dateLabel', () => {
  test.each([
    { now: '2026-10-20T01:10:00Z', want: 'Tuesday 20 October' },
    // 00:30 on the 20th in Tokyo, the tests' time zone, is still the 19th in UTC.
    { now: '2026-10-19T15:30:00Z', want: 'Tuesday 20 October' },
    { now: '2026-10-19T14:59:00Z', want: 'Monday 19 October' },
  ])('shows $now as "$want"', ({ now, want }) => {
    expect(dateLabel(new Date(now))).toBe(want);
  });
});

describe('greeting', () => {
  test.each([
    { time: '00:00', want: 'Good evening' },
    { time: '04:59', want: 'Good evening' },
    { time: '05:00', want: 'Good morning' },
    { time: '11:59', want: 'Good morning' },
    { time: '12:00', want: 'Good afternoon' },
    { time: '17:59', want: 'Good afternoon' },
    { time: '18:00', want: 'Good evening' },
    { time: '23:59', want: 'Good evening' },
  ])('says "$want" at $time', ({ time, want }) => {
    const [hours, minutes] = time.split(':').map(Number);
    expect(greeting(new Date(2026, 9, 20, hours, minutes))).toBe(want);
  });
});

describe('summary', () => {
  const item = (late: boolean) => ({ late });

  test.each([
    { items: [], want: 'All clear today.' },
    { items: [item(false)], want: '1 thing needs you today.' },
    { items: [item(true)], want: '1 thing needs you today. 1 is late.' },
    { items: [item(false), item(false)], want: '2 things need you today.' },
    { items: [item(true), item(false), item(false)], want: '3 things need you today. 1 is late.' },
    {
      items: [item(true), item(true), item(false), item(false), item(false), item(false)],
      want: '6 things need you today. 2 are late.',
    },
  ])('says "$want"', ({ items, want }) => {
    expect(summary(items)).toBe(want);
  });
});
