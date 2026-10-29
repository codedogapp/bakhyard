import { fireEvent, render, screen, within } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import type { Snapshot, Ticket } from '../jira/tickets';
import type { Report } from './report';
import TimePage from './TimePage.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

type Answer = () => Promise<Response>;

/** Answers the page's requests: the report's, then my open tickets'; nothing else. */
function answer(report: Answer, tickets: Answer) {
  const fetchMock = vi.fn<typeof fetch>(async (url) => (url === '/api/time/report' ? report() : tickets()));
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

function ticket(key: string, summary: string): Ticket {
  return { key, summary, status: 'To Do', category: 'to-do', url: `https://jira.example.com/browse/${key}` };
}

const searched = '2026-10-21T00:38:00Z';
// Monday 19 to Wednesday 21 October 2026, with nothing logged on the Tuesday.
const report: Report = {
  days: [
    { date: '2026-10-19', loggedHours: 8, missing: false },
    { date: '2026-10-20', loggedHours: 0, missing: true },
    { date: '2026-10-21', loggedHours: 1.5, missing: false },
  ],
  week: { loggedHours: 9.5, targetHours: 40 },
  searched,
};
const mine: Snapshot = {
  tickets: [ticket('WEB-12', 'Show the order history'), ticket('WEB-9', 'Fix the date picker in Safari')],
  searched,
};

const json = (body: Report | Snapshot) => async () => Response.json(body);
const status = (code: number) => async () => new Response('No\n', { status: code });
const loading = () => Promise.withResolvers<Response>().promise;

const select = (name: string) => screen.getByRole<HTMLSelectElement>('combobox', { name });
const options = (name: string) => [...select(name).options].map(({ text }) => text);
const dialog = () => screen.queryByRole('alertdialog');

/** The page with the form shown. */
async function filling() {
  const fetchMock = answer(json(report), json(mine));
  render(TimePage);
  await screen.findByRole('combobox', { name: 'Day' });
  return fetchMock;
}

test('shows the form under the head, over this week, once the report and my open tickets load', async () => {
  const fetchMock = answer(json(report), json(mine));
  render(TimePage);

  const day = await screen.findByRole('combobox', { name: 'Day' });
  const head = screen.getByRole('heading', { level: 1, name: 'Time' });
  const week = screen.getByRole('heading', { level: 2, name: 'This week' });
  expect(head.compareDocumentPosition(day)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  expect(day.compareDocumentPosition(week)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  expect(screen.getAllByRole('combobox').map((each) => each.getAttribute('aria-label'))).toEqual([
    'Day',
    'Ticket',
    'Hours',
  ]);
  expect(screen.getByRole('button', { name: 'Log' })).toHaveAttribute('type', 'submit');
  expect(screen.getByRole('status').textContent).toBe('');
  expect(fetchMock.mock.calls).toEqual([['/api/time/report'], ['/api/todos/tickets']]);
});

test("offers the report's days, my open tickets and the hours, and starts on the missing day and 8h", async () => {
  await filling();

  expect(options('Day')).toEqual(['Monday 19 Oct', 'Tuesday 20 Oct', 'Today']);
  expect(select('Day')).toHaveValue('2026-10-20');
  expect(options('Ticket')).toEqual([
    'Pick a ticket',
    'WEB-12 · Show the order history',
    'WEB-9 · Fix the date picker in Safari',
  ]);
  expect(select('Ticket')).toHaveValue('');
  expect(select('Ticket')).toBeRequired();
  expect(options('Hours')).toHaveLength(48);
  expect(options('Hours').slice(0, 2)).toEqual(['0h 15m', '0h 30m']);
  expect(select('Hours')).toHaveDisplayValue('8h');
});

test('starts on today when no day misses its hours', async () => {
  const logged = report.days?.map((day) => ({ ...day, missing: false }));
  answer(json({ ...report, days: logged }), json(mine));
  render(TimePage);

  expect(await screen.findByRole('combobox', { name: 'Day' })).toHaveDisplayValue('Today');
});

test('opens nothing on Log with no ticket picked', async () => {
  await filling();

  await fireEvent.submit(select('Day'));

  expect(dialog()).not.toBeInTheDocument();
});

test('asks in a dialog whether to log the hours on the ticket on the day picked', async () => {
  await filling();

  await fireEvent.change(select('Day'), { target: { value: '2026-10-19' } });
  await fireEvent.change(select('Ticket'), { target: { value: 'WEB-9' } });
  await fireEvent.change(select('Hours'), { target: { value: '450' } });
  await fireEvent.click(screen.getByRole('button', { name: 'Log' }));

  const asking = screen.getByRole('alertdialog', { name: 'Log these hours?' });
  expect(asking).toHaveAccessibleDescription(
    '7h 30m on WEB-9 · Fix the date picker in Safari, from 09:00 on Monday 19 Oct.'
  );
  expect(
    within(asking)
      .getAllByRole('button')
      .map(({ textContent }) => textContent)
  ).toEqual(['Cancel', 'Log']);
  expect(within(asking).getByRole('button', { name: 'Cancel' })).toHaveAttribute('autofocus');
});

test('closes on Cancel with no log', async () => {
  const fetchMock = await filling();
  await fireEvent.change(select('Ticket'), { target: { value: 'WEB-12' } });
  await fireEvent.click(screen.getByRole('button', { name: 'Log' }));

  await fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

  expect(dialog()).not.toBeInTheDocument();
  expect(fetchMock).toHaveBeenCalledTimes(2);
  expect(screen.getByRole('status').textContent).toBe('');
});

test('shows no form, and says nothing, while my open tickets load', async () => {
  answer(json(report), loading);
  render(TimePage);

  await screen.findByRole('heading', { level: 2, name: 'This week' });

  expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
  expect(screen.queryByText(/Jira tickets/)).not.toBeInTheDocument();
});

test.each([
  { name: "can't load", tickets: status(502), says: "Couldn't load your Jira tickets." },
  { name: "can't load without Jira", tickets: status(404), says: "Couldn't load your Jira tickets." },
  {
    name: "can't load before Jira's first search",
    tickets: json({ failure: 'search: 401 Unauthorized' }),
    says: "Couldn't load your Jira tickets.",
  },
  { name: 'are none', tickets: json({ tickets: [], searched }), says: 'No open Jira tickets to log hours on.' },
])('shows no form, and says so, when my open tickets $name', async ({ tickets, says }) => {
  answer(json(report), tickets);
  render(TimePage);

  const note = await screen.findByText(says);

  expect(note).toHaveClass('note');
  expect(note.compareDocumentPosition(screen.getByRole('heading', { name: 'This week' }))).toBe(
    Node.DOCUMENT_POSITION_FOLLOWING
  );
  expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
});

// My open tickets fail at once, so a line about them would show by the time the report's answer does.
test.each([
  { name: 'loads no day', report: json({ failure: 'find me: 401 Unauthorized' }), says: /^The last search failed/ },
  { name: "can't load", report: status(502), says: "Couldn't load the time report." },
  { name: 'is off', report: status(404), says: "Jira isn't set up." },
])('shows neither form nor line about my tickets when the report $name', async ({ report: answered, says }) => {
  answer(answered, status(502));
  render(TimePage);

  await screen.findByText(says);

  expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
  expect(screen.queryByText(/Jira tickets/)).not.toBeInTheDocument();
});
