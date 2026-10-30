import { render, screen, within } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import type { Item } from './api';
import YourApps from './YourApps.svelte';

const items: Item[] = [
  { id: 'time/1', app: 'time', title: "Log today's hours", source: 'Jira', late: false },
  { id: 'pulls/1', app: 'pulls', title: 'Nudge reviewers', source: 'Bitbucket', late: true },
  { id: 'time/2', app: 'time', title: 'Jira token expires', source: 'Jira', late: false },
];

function card(name: string): HTMLElement {
  const link = screen.getByRole('heading', { level: 3, name }).closest('a');
  expect(link).not.toBeNull();
  return link as HTMLElement;
}

test('links a numbered card to every app, in order', () => {
  render(YourApps, { props: { items } });

  expect(screen.getByRole('heading', { level: 2, name: 'Your apps' })).toBeInTheDocument();
  const cards = screen.getAllByRole('link');
  expect(
    cards.map((link) => [
      within(link).getByText(/^\d\d$/).textContent,
      within(link).getByRole('heading', { level: 3 }).textContent,
      link.getAttribute('href'),
    ])
  ).toEqual([
    ['01', 'Notes', '/notes'],
    ['02', 'Todos', '/todos'],
    ['03', 'Pull requests', '/pulls'],
    ['04', 'Time', '/time'],
    ['05', 'GitHub', '/github'],
    ['06', 'Games', '/games'],
    ['07', 'Planner', '/planner'],
    ['08', 'Nudge', '/nudge'],
    ['09', 'Alerts', '/alerts'],
  ]);
  expect(within(card('Time')).getByText('Hours logged on the Jira tickets')).toBeInTheDocument();
});

test.each([
  { name: 'Time', value: '2', noun: 'items', zero: false },
  { name: 'Pull requests', value: '1', noun: 'item', zero: false },
  { name: 'Notes', value: '0', noun: 'items', zero: true },
])("shows $value $noun on $name's card", ({ name, value, noun, zero }) => {
  render(YourApps, { props: { items } });

  const count = within(card(name)).getByText(value).parentElement;
  expect(count).toHaveTextContent(`${value} ${noun}`);
  expect(count?.classList.contains('zero')).toBe(zero);
});

test('shows the cards without counts before the Home loads', () => {
  render(YourApps);

  expect(screen.getAllByRole('link')).toHaveLength(9);
  expect(screen.queryByText(/^items?$/)).not.toBeInTheDocument();
});
