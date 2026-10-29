import { fireEvent, render, screen, within } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import type { Snapshot } from '../jira/tickets';
import type { Report } from './report';
import TimePage from './TimePage.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

type Answer = () => Promise<Response>;

/** Answers the page's requests: the report's, my open tickets' and the log's. */
function answer(report: Answer, log: Answer) {
  const fetchMock = vi.fn<typeof fetch>(async (url) => {
    switch (url) {
      case '/api/time/report':
        return report();
      case '/api/todos/tickets':
        return Response.json(mine);
      default:
        return log();
    }
  });
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const searched = '2026-10-21T00:38:00Z';
const mine: Snapshot = {
  tickets: [
    {
      key: 'WEB-9',
      summary: 'Fix the date picker in Safari',
      status: 'In Progress',
      category: 'in-progress',
      url: 'https://jira.example.com/browse/WEB-9',
    },
  ],
  searched,
};
// Monday 19 to Wednesday 21 October 2026, with nothing logged on the Tuesday, then with its 8 hours logged.
const report: Report = {
  days: [
    { date: '2026-10-19', loggedHours: 8, missing: false },
    { date: '2026-10-20', loggedHours: 0, missing: true },
    { date: '2026-10-21', loggedHours: 1.5, missing: false },
  ],
  week: { loggedHours: 9.5, targetHours: 40 },
  searched,
};
const filled: Report = {
  ...report,
  days: report.days?.map((day) => (day.missing ? { ...day, loggedHours: 8, missing: false } : day)),
  week: { loggedHours: 17.5, targetHours: 40 },
};

const json = (body: Report) => async () => Response.json(body);
const created = async () => new Response(null, { status: 201 });
/** The report, then the filled one each time after. */
const reports = () => vi.fn<Answer>(json(filled)).mockImplementationOnce(json(report));

const select = (name: string) => screen.getByRole<HTMLSelectElement>('combobox', { name });
const logButton = () => screen.getAllByRole('button', { name: 'Log' })[0];
const status = () => screen.getByRole('status');
const rows = () => screen.queryAllByRole('listitem').map((row) => row.textContent?.replaceAll(/\s+/g, ' ').trim());

/** Picks WEB-9 on the form, and Log on the form, then in the dialog. */
async function log(): Promise<void> {
  await screen.findByRole('combobox', { name: 'Day' });
  await fireEvent.change(select('Ticket'), { target: { value: 'WEB-9' } });
  await fireEvent.click(logButton());
  await fireEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Log' }));
}

test('logs the hours on the ticket on the day, then shows the report again and says so', async () => {
  const fetchMock = answer(reports(), created);
  render(TimePage);

  await log();

  expect(await screen.findByText('Logged')).toBe(status());
  expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
    '/api/time/report',
    '/api/todos/tickets',
    '/api/time/worklogs',
    '/api/time/report',
  ]);
  expect(fetchMock.mock.calls[2]?.[1]?.body).toBe('{"ticket":"WEB-9","date":"2026-10-20","minutes":480}');
  expect(screen.getByText('17h 30m / 40h')).toBeInTheDocument();
  expect(rows()).toEqual(['19 Oct Monday 8h', '20 Oct Tuesday 8h', '21 Oct Today 1h 30m']);
  expect([select('Day'), select('Ticket'), select('Hours')].map(({ value }) => value)).toEqual([
    '2026-10-20',
    'WEB-9',
    '480',
  ]);
});

test('says "Logging…" and logs nothing more while the log runs, with the form free', async () => {
  const held = Promise.withResolvers<Response>();
  const fetchMock = answer(reports(), () => held.promise);
  render(TimePage);

  await log();

  expect(status()).toHaveTextContent('Logging…');
  expect(logButton()).toHaveAttribute('aria-disabled', 'true');
  await fireEvent.click(logButton());
  expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  // The closed dialog's Log, which no one can reach in a browser, adds nothing either.
  await fireEvent.click(within(screen.getByRole('alertdialog', { hidden: true })).getByText('Log'));
  expect(select('Day')).toBeEnabled();
  held.resolve(new Response(null, { status: 201 }));
  expect(await screen.findByText('Logged')).toBe(status());
  expect(logButton()).toHaveAttribute('aria-disabled', 'false');
  expect(fetchMock.mock.calls.filter(([url]) => url === '/api/time/worklogs')).toHaveLength(1);
});

test.each([
  {
    name: "the service's reason when it refuses the log",
    log: async () => new Response('invalid date: not one of the last 7 days\n', { status: 400 }),
    says: 'invalid date: not one of the last 7 days',
  },
  {
    name: 'that the log failed on any other answer',
    log: async () => new Response(null, { status: 502 }),
    says: 'Log failed',
  },
  {
    name: "that the log failed when it can't be sent",
    log: () => Promise.reject(new TypeError('Failed to fetch')),
    says: 'Log failed',
  },
])('says $name, and keeps the report', async ({ log: answered, says }) => {
  const fetchMock = answer(reports(), answered);
  render(TimePage);

  await log();

  expect(await screen.findByText(says)).toBe(status());
  expect(fetchMock.mock.calls.filter(([url]) => url === '/api/time/report')).toHaveLength(1);
  expect(rows()).toContain('20 Oct Tuesday Missing 0h');
  expect(logButton()).toHaveAttribute('aria-disabled', 'false');
});

test("says the hours are logged, and keeps the report, when it can't load the report again", async () => {
  const report502 = vi.fn<Answer>(async () => new Response(null, { status: 502 })).mockImplementationOnce(json(report));
  answer(report502, created);
  render(TimePage);

  await log();

  expect(await screen.findByText('Logged')).toBe(status());
  expect(rows()).toContain('20 Oct Tuesday Missing 0h');
});

test("starts on the report's default day again once the day logged has left it", async () => {
  // The days moved on while the log ran: Tuesday 20 to Thursday 22, with nothing logged on the Wednesday.
  const next: Report = {
    ...filled,
    days: [
      { date: '2026-10-20', loggedHours: 8, missing: false },
      { date: '2026-10-21', loggedHours: 0, missing: true },
      { date: '2026-10-22', loggedHours: 0, missing: false },
    ],
  };
  answer(vi.fn<Answer>(json(next)).mockImplementationOnce(json(report)), created);
  render(TimePage);
  await screen.findByRole('combobox', { name: 'Day' });
  await fireEvent.change(select('Day'), { target: { value: '2026-10-19' } });

  await log();

  expect(await screen.findByText('Logged')).toBe(status());
  expect(select('Day')).toHaveValue('2026-10-21');
});

// jsdom's dialog fires no close event (test-setup.ts): the test fires the one a browser would.
test.each([
  { name: 'to Log where the browser lost it', focus: () => undefined, focused: logButton },
  { name: 'where the browser put it back', focus: () => select('Hours').focus(), focused: () => select('Hours') },
])('leaves the focus $name as the dialog closes', async ({ focus, focused }) => {
  answer(reports(), created);
  render(TimePage);
  await screen.findByRole('combobox', { name: 'Day' });
  await fireEvent.change(select('Ticket'), { target: { value: 'WEB-9' } });
  await fireEvent.click(logButton());
  const dialog = screen.getByRole('alertdialog');
  await fireEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }));

  focus();
  dialog.dispatchEvent(new Event('close'));

  expect(document.activeElement).toBe(focused());
});
