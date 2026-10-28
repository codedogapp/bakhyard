import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import type { Home } from './api';
import HomePage from './HomePage.svelte';

// 10:00 on Tuesday 20 October in Tokyo, the tests' time zone.
const now = new Date('2026-10-20T01:00:00Z');

const withoutWeek: Home = {
  name: 'Ada',
  items: [{ id: 'pulls/1', app: 'pulls', title: 'Nudge reviewers', source: 'Bitbucket', late: false }],
};

const withWeek: Home = { ...withoutWeek, week: { loggedHours: 31.5, targetHours: 40 } };

// The week's head, its link and its hours; null for each that doesn't show.
function weekParts(): (HTMLElement | null)[] {
  return [
    screen.queryByRole('heading', { level: 2, name: 'This week' }),
    screen.queryByRole('link', { name: 'Day report' }),
    screen.queryByText('31h 30m / 40h'),
  ];
}

test('shows the week when the Home has one', () => {
  render(HomePage, { props: { now, home: withWeek } });

  expect(weekParts()).not.toContain(null);
});

test('shows a week with no hours and no target, even when nothing needs you', () => {
  render(HomePage, { props: { now, home: { name: 'Ada', items: [], week: { loggedHours: 0, targetHours: 0 } } } });

  expect(screen.getByRole('heading', { level: 2, name: 'This week' })).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Day report' })).toBeInTheDocument();
  expect(screen.getByText('0h / 0h')).toBeInTheDocument();
});

test('ends with the week, after the apps', () => {
  render(HomePage, { props: { now, home: withWeek } });

  const heads = screen.getAllByRole('heading', { level: 2 }).map((head) => head.textContent);
  expect(heads).toEqual(['Needs you today', 'Your apps', 'This week']);
});

test('leaves out the week when the Home has none, and shows the rest as with one', () => {
  render(HomePage, { props: { now, home: withoutWeek } });

  expect(weekParts()).toEqual([null, null, null]);
  expect(screen.getByRole('heading', { level: 1, name: 'Good morning, Ada.' })).toBeInTheDocument();
  expect(screen.getByText('1 thing needs you today.')).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 2, name: 'Needs you today' })).toBeInTheDocument();
  expect(screen.getByRole('link', { name: /Nudge reviewers/ })).toHaveAttribute('href', '/pulls');
  expect(screen.getByRole('heading', { level: 2, name: 'Your apps' })).toBeInTheDocument();
  expect(screen.getAllByText('item')).toHaveLength(1);
});

test('takes the week away when a later Home has none, and brings it back with one', async () => {
  const { rerender } = render(HomePage, { props: { now, home: withWeek } });

  await rerender({ home: withoutWeek });

  expect(weekParts()).toEqual([null, null, null]);

  await rerender({ home: withWeek });

  expect(weekParts()).not.toContain(null);
});

test('shows no week before the Home first loads', () => {
  render(HomePage, { props: { now } });

  expect(weekParts()).toEqual([null, null, null]);
  expect(screen.getByRole('heading', { level: 2, name: 'Your apps' })).toBeInTheDocument();
});
