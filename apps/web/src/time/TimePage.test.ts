import { render, screen, within } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import type { Report } from './report';
import TimePage from './TimePage.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

function answer(response: () => Promise<Response>) {
  const fetchMock = vi.fn<typeof fetch>(response);
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const searched = '2026-10-21T00:38:00Z';
const report: Report = {
  days: [
    { date: '2026-10-19', loggedHours: 8, missing: false },
    { date: '2026-10-20', loggedHours: 0, missing: true },
    { date: '2026-10-21', loggedHours: 1.5, missing: false },
  ],
  week: { loggedHours: 9.5, targetHours: 40 },
  searched,
};

const loaded = (snapshot: Report) => async () => Response.json(snapshot);
const loading = () => Promise.withResolvers<Response>().promise;

const heads = () => screen.queryAllByRole('heading', { level: 2 }).map((head) => head.textContent);
const rows = () => screen.queryAllByRole('listitem').map((row) => row.textContent?.replaceAll(/\s+/g, ' ').trim());

test('loads the time report as it shows, with this week and the last 7 days', async () => {
  const fetchMock = answer(loaded(report));
  render(TimePage);

  expect(screen.getByRole('heading', { level: 1, name: 'Time' })).toBeInTheDocument();
  expect(await screen.findByText('9h 30m / 40h')).toBeInTheDocument();
  expect(heads()).toEqual(['This week', 'Last 7 days']);
  expect(fetchMock.mock.calls).toEqual([['/api/time/report']]);
});

test('lists each day with its date, weekday and hours, and marks the missing one', async () => {
  answer(loaded(report));
  render(TimePage);

  expect(await screen.findByText('1 day missing')).toBeInTheDocument();
  expect(rows()).toEqual(['19 Oct Monday 8h', '20 Oct Tuesday Missing 0h', '21 Oct Today 1h 30m']);
  expect(screen.getByText('20 Oct')).toHaveAttribute('datetime', '2026-10-20');
  expect(screen.getByText('Missing')).toHaveClass('chip', 'chip-filled');
  const hours = within(screen.getByRole('list')).getAllByText(/^\d+h/);
  expect(hours.map((day) => day.classList.contains('logged'))).toEqual([true, false, true]);
});

test('notes no missing day when none is', async () => {
  answer(loaded({ ...report, days: report.days?.map((day) => ({ ...day, missing: false })) }));
  render(TimePage);

  expect(await screen.findByText('20 Oct')).toBeInTheDocument();
  expect(screen.queryByText(/missing/i)).not.toBeInTheDocument();
});

test('shows nothing under its head while the report loads', () => {
  answer(loading);
  render(TimePage);

  expect(screen.getByRole('heading', { level: 1, name: 'Time' })).toBeInTheDocument();
  expect(heads()).toEqual([]);
  expect(screen.queryByRole('paragraph')).not.toBeInTheDocument();
});

test.each([
  { name: "Jira isn't set up", status: 404, says: "Jira isn't set up." },
  { name: "the report couldn't load", status: 502, says: "Couldn't load the time report." },
])('says so, with no section, when $name', async ({ status, says }) => {
  answer(async () => new Response('No\n', { status }));
  render(TimePage);

  expect(await screen.findByText(says)).toHaveClass('off');
  expect(heads()).toEqual([]);
});

test('says why the last search failed, and when the hours it shows are from', async () => {
  answer(loaded({ ...report, failure: 'search worklogs: 502 Bad Gateway' }));
  render(TimePage);

  const line = await screen.findByText(/^The last search failed/);
  expect(line).toHaveTextContent(
    'The last search failed: search worklogs: 502 Bad Gateway. These hours are from 21 October at 09:38.'
  );
  expect(line).toHaveClass('failure');
  expect(within(line).getByText('21 October at 09:38')).toHaveAttribute('datetime', searched);
  expect(heads()).toEqual(['This week', 'Last 7 days']);
});

test('says why the last search failed, with no section, before any search has succeeded', async () => {
  answer(loaded({ failure: 'find me: 401 Unauthorized' }));
  render(TimePage);

  const line = await screen.findByText(/^The last search failed/);
  expect(line).toHaveTextContent(/^The last search failed: find me: 401 Unauthorized\.$/);
  expect(line).toHaveClass('off');
  expect(heads()).toEqual([]);
});
