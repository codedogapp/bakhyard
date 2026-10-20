import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import type { Home } from './api';
import Hero from './Hero.svelte';

const home: Home = {
  name: 'Ada',
  items: [
    { id: 'pulls/1', app: 'pulls', title: 'Nudge reviewers', source: 'Bitbucket', late: true },
    { id: 'github/1', app: 'github', title: 'Review requested', source: 'GitHub', late: false },
  ],
  week: { loggedHours: 31.5, targetHours: 40 },
};
// 11:59 on Tuesday 20 October in Tokyo, the tests' time zone.
const now = new Date('2026-10-20T02:59:00Z');

test('greets the user by name, with the date and a summary', () => {
  render(Hero, { props: { now, home } });

  expect(screen.getByText('Tuesday 20 October')).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 1, name: 'Good morning, Ada.' })).toBeInTheDocument();
  expect(screen.getByText('Ada').tagName).toBe('EM');
  expect(screen.getByText('2 things need you today. 1 is late.')).toBeInTheDocument();
});

test('greets with no name and no summary until the Home loads', () => {
  render(Hero, { props: { now } });

  expect(screen.getByRole('heading', { level: 1, name: 'Good morning.' })).toBeInTheDocument();
  expect(screen.queryByText(/today/)).not.toBeInTheDocument();
});

test.each([
  { at: '2026-10-20T03:00:00Z', date: 'Tuesday 20 October', want: 'Good afternoon, Ada.' },
  { at: '2026-10-20T15:00:00Z', date: 'Wednesday 21 October', want: 'Good evening, Ada.' },
])('shows "$date" and "$want" at $at', ({ at, date, want }) => {
  render(Hero, { props: { now: new Date(at), home } });

  expect(screen.getByText(date)).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 1, name: want })).toBeInTheDocument();
});
