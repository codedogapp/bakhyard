import { render, screen, within } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import JiraTickets from './JiraTickets.svelte';
import type { Ticket, Tickets } from './tickets';

const review: Ticket = {
  key: 'WEB-12',
  summary: 'Show the <b>order</b> history',
  status: 'In Review',
  category: 'in-progress',
  url: 'https://jira.example.com/browse/WEB-12',
};
const certificates: Ticket = {
  key: 'OPS-31',
  summary: 'Rotate the staging certificates',
  status: 'To Do',
  category: 'to-do',
  url: 'https://jira.example.com/browse/OPS-31',
};
const searched = '2026-10-26T00:41:00Z';

function show(tickets: Tickets) {
  return render(JiraTickets, { props: { tickets } });
}

test('lists the tickets in their order under a head that counts them, each a link to the ticket', () => {
  show({ kind: 'loaded', snapshot: { tickets: [review, certificates], searched } });

  expect(screen.getByRole('heading', { level: 2, name: 'Jira tickets' })).toBeInTheDocument();
  expect(screen.getByText('2 tickets')).toBeInTheDocument();
  const links = within(screen.getByRole('list')).getAllByRole('link');
  expect(links.map((link) => [link.getAttribute('href'), link.textContent?.replaceAll(/\s+/g, ' ').trim()])).toEqual([
    ['https://jira.example.com/browse/WEB-12', 'WEB-12 Show the <b>order</b> history In Review'],
    ['https://jira.example.com/browse/OPS-31', 'OPS-31 Rotate the staging certificates To Do'],
  ]);
  expect(screen.queryByText(/failed/)).not.toBeInTheDocument();
});

test.each<[Ticket['category'], boolean]>([
  ['in-progress', true],
  ['to-do', false],
  ['done', false],
  ['unknown', false],
])('fills the status chip of a ticket in the %s category: %s', (category, filled) => {
  show({ kind: 'loaded', snapshot: { tickets: [{ ...review, category }], searched } });

  const chip = screen.getByText('In Review');
  expect(chip).toHaveClass('chip');
  expect(chip.classList.contains('chip-filled')).toBe(filled);
});

test('says when there are no open tickets, with no count', () => {
  show({ kind: 'loaded', snapshot: { tickets: [], searched } });

  expect(screen.getByRole('heading', { level: 2, name: 'Jira tickets' })).toBeInTheDocument();
  expect(screen.getByText('No open tickets.')).toBeInTheDocument();
  expect(screen.queryByText(/^\d+ tickets?$/)).not.toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
});

test('says why the last search failed, and when the tickets it shows are from', () => {
  const failure = 'search Jira: 502 Bad Gateway';
  show({ kind: 'loaded', snapshot: { tickets: [review], searched, failure } });

  const line = screen.getByText(/^The last search failed/);
  expect(line).toHaveTextContent(
    'The last search failed: search Jira: 502 Bad Gateway. These tickets are from 26 October at 09:41.'
  );
  expect(within(line).getByText('26 October at 09:41')).toHaveAttribute('datetime', searched);
  expect(screen.getByText('1 ticket')).toBeInTheDocument();
  expect(within(screen.getByRole('list')).getByRole('link')).toHaveAttribute('href', review.url);
});

test('says why the last search failed, with no list, before any search has succeeded', () => {
  show({ kind: 'loaded', snapshot: { failure: 'search Jira: 401 Unauthorized' } });

  expect(screen.getByText(/^The last search failed/)).toHaveTextContent(
    /^The last search failed: search Jira: 401 Unauthorized\.$/
  );
  expect(screen.queryByText('No open tickets.')).not.toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
  expect(screen.queryByText(/^\d+ tickets?$/)).not.toBeInTheDocument();
});

test('says when the tickets could not load', () => {
  show({ kind: 'failed' });

  expect(screen.getByRole('heading', { level: 2, name: 'Jira tickets' })).toBeInTheDocument();
  expect(screen.getByText("Couldn't load the tickets.")).toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
});

test.each<Tickets>([{ kind: 'loading' }, { kind: 'off' }])('shows nothing while $kind', (tickets) => {
  const { container } = show(tickets);

  expect(container.childElementCount).toBe(0);
  expect(container.textContent).toBe('');
});
