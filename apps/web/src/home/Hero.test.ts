import { act, render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import Hero from './Hero.svelte';

const home = { name: 'Ada', items: [{ late: true }, { late: false }] };

beforeEach(() => {
  vi.useFakeTimers();
  // 11:59:59 on Tuesday 20 October in Tokyo, the tests' time zone.
  vi.setSystemTime(new Date('2026-10-20T02:59:59Z'));
});

afterEach(() => {
  vi.useRealTimers();
});

test('greets the user by name, with the date and a summary', () => {
  render(Hero, { props: { home } });

  expect(screen.getByText('Tuesday 20 October')).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 1, name: 'Good morning, Ada.' })).toBeInTheDocument();
  expect(screen.getByText('Ada').tagName).toBe('EM');
  expect(screen.getByText('2 things need you today. 1 is late.')).toBeInTheDocument();
});

test('greets with no name and no summary until the Home loads', () => {
  render(Hero);

  expect(screen.getByRole('heading', { level: 1, name: 'Good morning.' })).toBeInTheDocument();
  expect(screen.queryByText(/today/)).not.toBeInTheDocument();
});

test.each([
  { now: '2026-10-20T02:59:59Z', date: 'Tuesday 20 October', want: 'Good afternoon, Ada.' },
  { now: '2026-10-20T14:59:59Z', date: 'Wednesday 21 October', want: 'Good evening, Ada.' },
])('moves on with the clock to "$date" and "$want"', async ({ now, date, want }) => {
  vi.setSystemTime(new Date(now));
  render(Hero, { props: { home } });

  await act(() => vi.advanceTimersByTime(1000));

  expect(screen.getByText(date)).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 1, name: want })).toBeInTheDocument();
});

test('stops its clock when it goes away', () => {
  const { unmount } = render(Hero, { props: { home } });
  expect(vi.getTimerCount()).toBe(1);

  unmount();

  expect(vi.getTimerCount()).toBe(0);
});
