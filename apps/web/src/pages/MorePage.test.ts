import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import type { Item } from '../home/api';
import MorePage from './MorePage.svelte';

const items: Item[] = [
  { id: 'time/1', app: 'time', title: "Log today's hours", source: 'Jira', late: false },
  { id: 'pulls/1', app: 'pulls', title: 'Nudge reviewers', source: 'Bitbucket', late: true },
  { id: 'time/2', app: 'time', title: 'Jira token expires', source: 'Jira', late: false },
  { id: 'github/1', app: 'github', title: 'Review requested', source: 'GitHub', late: false },
];

function cardTexts(): string[] {
  return screen.getAllByRole('link').map((card) => (card.textContent ?? '').replace(/\s+/g, ' ').trim());
}

test('lists the apps without a tab of their own as cards, with their counts', () => {
  render(MorePage, { props: { items } });

  expect(screen.getByRole('heading', { level: 1, name: 'More' })).toBeInTheDocument();
  expect(screen.getByText('The apps without a tab of their own.')).toBeInTheDocument();
  // Under the page's h1, the apps' names are h2s.
  expect(screen.getAllByRole('heading', { level: 2 }).map((h) => h.textContent)).toEqual([
    'Time',
    'GitHub',
    'Games',
    'Planner',
    'Nudge',
  ]);
  expect(screen.getAllByRole('link').map((card) => card.getAttribute('href'))).toEqual([
    '/time',
    '/github',
    '/games',
    '/planner',
    '/nudge',
  ]);
  expect(cardTexts()).toEqual([
    '04 2 items Time Hours logged on the Jira tickets',
    '05 1 item GitHub PRs, reviews and issues across owners',
    '06 0 items Games Backloggd and Exophase',
    '07 0 items Planner Copilot plans saved to the vault',
    '08 0 items Nudge Questions from your agents, answered on your phone',
  ]);
});

test('shows the cards without counts before the Home loads', () => {
  render(MorePage);

  expect(cardTexts()).toEqual([
    '04 Time Hours logged on the Jira tickets',
    '05 GitHub PRs, reviews and issues across owners',
    '06 Games Backloggd and Exophase',
    '07 Planner Copilot plans saved to the vault',
    '08 Nudge Questions from your agents, answered on your phone',
  ]);
});
