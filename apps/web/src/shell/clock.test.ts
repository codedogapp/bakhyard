import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';

import { clockLabel, startClock } from './clock';

describe('clockLabel', () => {
  test.each([
    { now: '2026-10-20T01:12:30Z', timeZone: 'UTC', want: '01:12 · UTC' },
    { now: '2026-10-20T15:05:00Z', timeZone: 'Asia/Tokyo', want: '00:05 · Tokyo' },
    { now: '2026-10-20T01:11:00Z', timeZone: 'America/Argentina/Buenos_Aires', want: '22:11 · Buenos Aires' },
  ])('shows $now in $timeZone as "$want"', ({ now, timeZone, want }) => {
    expect(clockLabel(new Date(now), timeZone)).toBe(want);
  });

  test('uses the local time zone by default', () => {
    expect(clockLabel(new Date('2026-10-20T01:11:59Z'))).toBe('10:11 · Tokyo');
  });
});

describe('startClock', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-10-20T01:11:59.500Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test('ticks at the start of every minute', () => {
    const ticks: string[] = [];
    const stop = startClock((now) => ticks.push(now.toISOString()));

    vi.advanceTimersByTime(499);
    expect(ticks).toEqual([]);

    vi.advanceTimersByTime(1);
    expect(ticks).toEqual(['2026-10-20T01:12:00.000Z']);

    vi.advanceTimersByTime(60_000);
    expect(ticks).toEqual(['2026-10-20T01:12:00.000Z', '2026-10-20T01:13:00.000Z']);

    stop();
  });

  test('realigns with the minute after a late tick', () => {
    const ticks: string[] = [];
    const stop = startClock((now) => ticks.push(now.toISOString()));

    // The page was throttled or the machine slept, so the first tick comes 20 s late.
    vi.setSystemTime(new Date('2026-10-20T01:12:20Z'));
    vi.advanceTimersByTime(500);
    expect(ticks).toEqual(['2026-10-20T01:12:20.500Z']);

    vi.advanceTimersByTime(39_500);
    expect(ticks).toEqual(['2026-10-20T01:12:20.500Z', '2026-10-20T01:13:00.000Z']);

    stop();
  });

  test('stops ticking once stopped', () => {
    const onTick = vi.fn();
    const stop = startClock(onTick);

    vi.advanceTimersByTime(60_500);
    stop();
    vi.advanceTimersByTime(120_000);

    expect(onTick).toHaveBeenCalledTimes(2);
    expect(vi.getTimerCount()).toBe(0);
  });
});
