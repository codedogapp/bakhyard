import { render, screen, within } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import type { Item } from './api';
import NeedsYou from './NeedsYou.svelte';

// 10:00 on Tuesday 20 October in Tokyo, the tests' time zone.
const now = new Date('2026-10-20T01:00:00Z');

const items: Item[] = [
  {
    id: 'pulls/1',
    app: 'pulls',
    title: 'Nudge reviewers',
    source: 'Bitbucket',
    due: '2026-10-19T23:00:00Z',
    late: true,
  },
  {
    id: 'todos/1',
    app: 'todos',
    title: 'Settlement report export',
    detail: 'Was due yesterday.',
    source: 'Jira',
    due: '2026-10-19T01:00:00Z',
    late: true,
  },
  { id: 'time/1', app: 'time', title: "Log today's hours", source: 'Jira', due: '2026-10-20T04:00:00Z', late: false },
  { id: 'github/1', app: 'github', title: 'Review requested: dotfiles#14', source: 'GitHub', late: false },
];

test('heads the list with the number of items', () => {
  render(NeedsYou, { props: { items, now } });

  expect(screen.getByRole('heading', { level: 2, name: 'Needs you today' })).toBeInTheDocument();
  expect(screen.getByText('4 items · most urgent first')).toBeInTheDocument();
});

test.each([
  { row: 0, href: '/pulls', source: 'Bitbucket', title: 'Nudge reviewers', due: '2 hours ago' },
  { row: 1, href: '/todos', source: 'Jira', title: 'Settlement report export', due: '1 day ago' },
  { row: 2, href: '/time', source: 'Jira', title: "Log today's hours", due: 'in 3 hours' },
  { row: 3, href: '/github', source: 'GitHub', title: 'Review requested: dotfiles#14', due: undefined },
])('shows "$title" in row $row, linking to $href', ({ row, href, source, title, due }) => {
  render(NeedsYou, { props: { items, now } });

  const link = within(screen.getAllByRole('listitem')[row]).getByRole('link');
  expect(link).toHaveAttribute('href', href);
  expect(within(link).getByText(source)).toBeInTheDocument();
  expect(within(link).getByText(title)).toBeInTheDocument();
  expect(link.querySelector('time')?.textContent).toBe(due);
});

test('marks only the late items with a Late chip', () => {
  render(NeedsYou, { props: { items, now } });

  const late = screen.getAllByRole('listitem').map((row) => within(row).queryByText('Late') !== null);
  expect(late).toEqual([true, true, false, false]);
});

test('shows a detail under its title', () => {
  render(NeedsYou, { props: { items, now } });

  const link = within(screen.getAllByRole('listitem')[1]).getByRole('link');
  expect(within(link).getByText('Was due yesterday.')).toBeInTheDocument();
});

test('marks each due time up with the time it stands for', () => {
  render(NeedsYou, { props: { items, now } });

  expect(screen.getByText('in 3 hours')).toHaveAttribute('datetime', '2026-10-20T04:00:00Z');
});

test('says nothing needs the user when there are no items', () => {
  render(NeedsYou, { props: { items: [], now } });

  expect(screen.getByRole('heading', { level: 2, name: 'Needs you today' })).toBeInTheDocument();
  expect(screen.getByText('Nothing needs you today.')).toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
  expect(screen.queryByText(/most urgent first/)).not.toBeInTheDocument();
});
