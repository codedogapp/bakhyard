import { act, render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import type { Connection } from '../home/poll';
import TopBar from './TopBar.svelte';

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date('2026-10-20T01:11:59Z'));
});

afterEach(() => {
  vi.useRealTimers();
});

test('shows the local time and city, and moves with the minute', async () => {
  render(TopBar, { props: { connection: 'online' } });
  expect(screen.getByText('10:11 · Tokyo')).toBeInTheDocument();

  await act(() => vi.advanceTimersByTime(1000));

  expect(screen.getByText('10:12 · Tokyo')).toBeInTheDocument();
});

test.each<{ connection: Connection; asOf?: Date; want: string }>([
  { connection: 'connecting', want: '' },
  { connection: 'online', want: 'Online' },
  { connection: 'offline', asOf: new Date('2026-10-20T01:10:00Z'), want: 'Offline · as of 10:10' },
])('says "$want" in its status when $connection', ({ connection, asOf, want }) => {
  render(TopBar, { props: { connection, asOf } });

  expect(screen.getByRole('status')).toHaveTextContent(want, { normalizeWhitespace: true });
});

test('stops the clock when it goes away', () => {
  const { unmount } = render(TopBar, { props: { connection: 'online' } });
  expect(vi.getTimerCount()).toBe(1);

  unmount();

  expect(vi.getTimerCount()).toBe(0);
});
