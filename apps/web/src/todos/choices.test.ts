import { expect, test } from 'vitest';

import { ticketChoices } from './choices';
import type { Ticket } from './tickets';

function ticket(key: string, summary: string): Ticket {
  return { key, summary, status: 'To Do', category: 'to-do', url: `https://jira.example.com/browse/${key}` };
}

/** My open tickets in Jira's order, which isn't the keys'. */
const open = new Map(
  [ticket('WEB-12', 'Show the order history'), ticket('WEB-9', 'Fix the date picker')].map((each) => [each.key, each])
);
const none = { key: '', label: 'No ticket' };
const mine = [
  { key: 'WEB-12', label: 'WEB-12 · Show the order history' },
  { key: 'WEB-9', label: 'WEB-9 · Fix the date picker' },
];

test.each([
  ['none, then my open tickets in their order, for a todo with no ticket', open, undefined, [none, ...mine]],
  ["the todo's own ticket once, when it's one of mine", open, 'WEB-9', [none, ...mine]],
  [
    "the todo's own ticket by its key alone right after none, when it isn't",
    open,
    'WEB-4',
    [none, { key: 'WEB-4', label: 'WEB-4' }, ...mine],
  ],
  ['none alone, for a todo with no ticket, without open tickets', new Map(), undefined, [none]],
  [
    "none and the todo's own ticket, without open tickets",
    new Map(),
    'WEB-4',
    [none, { key: 'WEB-4', label: 'WEB-4' }],
  ],
])('offers %s', (_, tickets, linked, choices) => {
  expect(ticketChoices(tickets, linked)).toEqual(choices);
});
