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

// home asked the time service for the week, and couldn't get it.
const unavailable: Home = { ...withoutWeek, weekState: 'unavailable' };

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

test("says the hours are unavailable when home couldn't get the week, under its head and link, after the apps", () => {
  render(HomePage, { props: { now, home: unavailable } });

  const [head, link, hours] = weekParts();
  expect(head).toBeInTheDocument();
  expect(link).toHaveAttribute('href', '/time');
  expect(hours).toBeNull();
  expect(screen.getByText('Hours unavailable')).toBeInTheDocument();
  const heads = screen.getAllByRole('heading', { level: 2 }).map((heading) => heading.textContent);
  expect(heads).toEqual(['Needs you today', 'Your apps', 'This week']);
});

test('shows the week when the Home has one, whatever its state says', () => {
  render(HomePage, { props: { now, home: { ...withWeek, weekState: 'unavailable' } } });

  expect(weekParts()).not.toContain(null);
  expect(screen.queryByText('Hours unavailable')).not.toBeInTheDocument();
});

test('leaves out the week when the time service had none to give, as without Jira', () => {
  render(HomePage, { props: { now, home: { ...withoutWeek, weekState: 'ok' } } });

  expect(weekParts()).toEqual([null, null, null]);
  expect(screen.queryByText('Hours unavailable')).not.toBeInTheDocument();
});

test('follows each later Home: from the hours to unavailable, to no week, and back', async () => {
  const { rerender } = render(HomePage, { props: { now, home: withWeek } });

  await rerender({ home: unavailable });

  expect(screen.getByText('Hours unavailable')).toBeInTheDocument();
  expect(screen.queryByText('31h 30m / 40h')).not.toBeInTheDocument();

  await rerender({ home: withoutWeek });

  expect(weekParts()).toEqual([null, null, null]);
  expect(screen.queryByText('Hours unavailable')).not.toBeInTheDocument();

  await rerender({ home: unavailable });

  expect(screen.getByText('Hours unavailable')).toBeInTheDocument();

  await rerender({ home: withWeek });

  expect(weekParts()).not.toContain(null);
  expect(screen.queryByText('Hours unavailable')).not.toBeInTheDocument();
});
