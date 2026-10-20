import { act, render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import TopBar from './TopBar.svelte';

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date('2026-10-20T01:11:59Z'));
});

afterEach(() => {
  vi.useRealTimers();
});

test('shows the local time and city, and moves with the minute', async () => {
  render(TopBar);
  expect(screen.getByText('10:11 · Tokyo')).toBeInTheDocument();

  await act(() => vi.advanceTimersByTime(1000));

  expect(screen.getByText('10:12 · Tokyo')).toBeInTheDocument();
});

test('stops the clock when it goes away', () => {
  const { unmount } = render(TopBar);
  expect(vi.getTimerCount()).toBe(1);

  unmount();

  expect(vi.getTimerCount()).toBe(0);
});
