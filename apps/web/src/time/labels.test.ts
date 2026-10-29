import { expect, test } from 'vitest';

import type { Ticket, Tickets } from '../jira/tickets';
import type { FillStatus } from './fill.svelte';
import { entryText, fillLabel, ticketsNote } from './labels';

const ticket: Ticket = {
  key: 'WEB-9',
  summary: 'Fix the date picker in Safari',
  status: 'In Progress',
  category: 'in-progress',
  url: 'https://example.atlassian.net/browse/WEB-9',
};
const searched = '2026-10-21T00:40:00Z';

test('says what the dialog asks to log', () => {
  expect(entryText({ date: '2026-10-16', ticket, minutes: 480 })).toBe(
    '8h on WEB-9 · Fix the date picker in Safari, from 09:00 on Friday 16 Oct.'
  );
  expect(entryText({ date: '2026-10-19', ticket, minutes: 45 })).toBe(
    '0h 45m on WEB-9 · Fix the date picker in Safari, from 09:00 on Monday 19 Oct.'
  );
});

test.each<{ status: FillStatus; want: string }>([
  { status: { kind: 'idle' }, want: '' },
  { status: { kind: 'logging' }, want: 'Logging…' },
  { status: { kind: 'logged' }, want: 'Logged' },
  {
    status: { kind: 'refused', reason: 'invalid date: not one of the last 7 days' },
    want: 'invalid date: not one of the last 7 days',
  },
  { status: { kind: 'failed' }, want: 'Log failed' },
])('says "$want" for a log that is $status.kind', ({ status, want }) => {
  expect(fillLabel(status)).toBe(want);
});

test.each<{ name: string; tickets: Tickets; want: string | undefined }>([
  { name: 'while they load', tickets: { kind: 'loading' }, want: undefined },
  { name: 'without Jira', tickets: { kind: 'off' }, want: "Couldn't load your Jira tickets." },
  { name: 'when they fail', tickets: { kind: 'failed' }, want: "Couldn't load your Jira tickets." },
  {
    name: 'before the first search',
    tickets: { kind: 'loaded', snapshot: { failure: 'Jira answered 503' } },
    want: "Couldn't load your Jira tickets.",
  },
  {
    name: 'when none is open',
    tickets: { kind: 'loaded', snapshot: { tickets: [], searched } },
    want: 'No open Jira tickets to log hours on.',
  },
  { name: 'with tickets', tickets: { kind: 'loaded', snapshot: { tickets: [ticket], searched } }, want: undefined },
])('notes my open tickets $name', ({ tickets, want }) => {
  expect(ticketsNote(tickets)).toBe(want);
});
