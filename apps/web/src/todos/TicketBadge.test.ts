import { render } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import TicketBadge from './TicketBadge.svelte';
import type { Ticket } from './tickets';

const picker: Ticket = {
  key: 'WEB-9',
  summary: 'Fix the date picker in Safari',
  status: 'In Progress',
  category: 'in-progress',
  url: 'https://jira.example.com/browse/WEB-9',
};

function show(ticket: Ticket | undefined, ticked = false) {
  const { container } = render(TicketBadge, { props: { key: 'WEB-9', ticket, ticked } });
  return container;
}

const text = (badge: HTMLElement) => badge.textContent?.replaceAll(/\s+/g, ' ').trim();

test("shows the key, then the ticket's status in a chip", () => {
  const badge = show(picker);

  expect(text(badge)).toBe('WEB-9 In Progress');
  expect(badge.querySelector('.chip')).toHaveTextContent(/^In Progress$/);
});

test('shows the key alone without the ticket', () => {
  const badge = show(undefined);

  expect(badge.textContent?.trim()).toBe('WEB-9');
  expect(badge.querySelector('.chip')).toBeNull();
});

test.each<[Ticket['category'], boolean]>([
  ['in-progress', true],
  ['to-do', false],
  ['done', false],
  ['unknown', false],
])('fills the chip of a ticket in the %s category: %s', (category, filled) => {
  const chip = show({ ...picker, category }).querySelector('.chip');

  expect(chip?.classList.contains('chip-filled')).toBe(filled);
});

const shipped: Ticket = { ...picker, status: 'Closed', category: 'done' };

test("offers to tick a todo whose ticket is done, after the ticket's status", () => {
  const badge = show(shipped);

  expect(text(badge)).toBe('WEB-9 Closed Done in Jira');
  expect(badge.querySelector('.chip + .offer')).toHaveTextContent(/^Done in Jira$/);
});

test('makes no offer once the todo is ticked', () => {
  const badge = show(shipped, true);

  expect(text(badge)).toBe('WEB-9 Closed');
  expect(badge.querySelector('.offer')).toBeNull();
});

test.each<Ticket['category']>(['to-do', 'in-progress', 'unknown'])(
  'makes no offer for a ticket in the %s category',
  (category) => {
    expect(show({ ...picker, category }).querySelector('.offer')).toBeNull();
  }
);
