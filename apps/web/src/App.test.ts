import { act, render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import App from './App.svelte';
import type { Home } from './home/api';

const home: Home = {
  name: 'Ada',
  items: [
    {
      id: 'pulls/1',
      app: 'pulls',
      title: 'Nudge reviewers',
      source: 'Bitbucket',
      due: '2026-10-20T00:59:59Z',
      late: true,
    },
    // 30 seconds after 11:59:59 in Tokyo, the tests' time zone: "in 1 minute" then, and "now" at 12:00.
    { id: 'time/1', app: 'time', title: "Log today's hours", source: 'Jira', due: '2026-10-20T03:00:29Z', late: false },
  ],
};

beforeEach(() => {
  vi.stubGlobal(
    'fetch',
    vi.fn<typeof fetch>(async () => Response.json(home))
  );
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

test('shows the frame and the apps, then the hero, the feed, the counts and the status once the Home loads', async () => {
  render(App);

  expect(screen.getByText(/ · Tokyo$/)).toBeInTheDocument();
  expect(screen.getByRole('banner')).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Home', current: 'page' })).toBeInTheDocument();
  expect(screen.queryByRole('heading', { level: 2, name: 'Needs you today' })).not.toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 2, name: 'Your apps' })).toBeInTheDocument();
  expect(screen.queryByText(/^items?$/)).not.toBeInTheDocument();
  expect(await screen.findByRole('heading', { level: 1, name: /^Good \w+, Ada\.$/ })).toBeInTheDocument();
  expect(screen.getByText('2 things need you today. 1 is late.')).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 2, name: 'Needs you today' })).toBeInTheDocument();
  expect(screen.getByText('Nudge reviewers')).toBeInTheDocument();
  expect(screen.getByText("Log today's hours")).toBeInTheDocument();
  expect(screen.getAllByText('item')).toHaveLength(2);
  expect(screen.getByRole('status')).toHaveTextContent('Online');
});

test('moves the clock, the greeting and the due times on together', async () => {
  vi.useFakeTimers();
  // 11:59:59 on Tuesday 20 October in Tokyo.
  vi.setSystemTime(new Date('2026-10-20T02:59:59Z'));
  render(App);
  await act(() => vi.advanceTimersByTimeAsync(0));

  expect(screen.getByText('11:59 · Tokyo')).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 1, name: 'Good morning, Ada.' })).toBeInTheDocument();
  expect(screen.getByText('in 1 minute')).toBeInTheDocument();

  await act(() => vi.advanceTimersByTimeAsync(1000));

  expect(screen.getByText('12:00 · Tokyo')).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 1, name: 'Good afternoon, Ada.' })).toBeInTheDocument();
  expect(screen.getByText('now')).toBeInTheDocument();
});

test('stops its clock and its polling when it goes away', async () => {
  vi.useFakeTimers();
  const { unmount } = render(App);
  await act(() => vi.advanceTimersByTimeAsync(0));
  expect(vi.getTimerCount()).toBe(2);

  unmount();

  expect(vi.getTimerCount()).toBe(0);
});
