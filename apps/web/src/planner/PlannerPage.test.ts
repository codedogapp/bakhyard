import { render, screen, within } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import type { Plan } from './api';
import PlannerPage from './PlannerPage.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

function answer(response: () => Promise<Response>) {
  const fetchMock = vi.fn<typeof fetch>(response);
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

// Saturday 31 October 2026, 09:41 in Tokyo, where the tests run.
const now = new Date('2026-10-31T09:41:00+09:00');
const plans: Plan[] = [
  {
    id: '0b6c1f9e-2d4a-4c3b-9a8e-5f7d6c4b3a21',
    title: 'Back up the home lab',
    started_at: '2026-10-30T23:50:00Z',
    updated_at: '2026-10-31T00:20:00.123456789Z',
  },
  {
    id: '7e5d4c3b-2a19-4f8e-8d7c-6b5a49382716',
    title: '',
    started_at: '2026-10-30T10:00:00Z',
    updated_at: '2026-10-30T11:00:00Z',
  },
  {
    id: 'c3b2a190-8f7e-4d6c-9b5a-493827160f5e',
    title: 'Move the blog to a static site',
    started_at: '2026-10-18T08:00:00Z',
    updated_at: '2026-10-19T08:00:00Z',
  },
];

const loaded = (list: Plan[]) => async () => Response.json({ plans: list });
const rows = () => within(screen.getByRole('list')).getAllByRole('listitem');

test('lists the plans as it shows, in the order they come, a row each, none a link', async () => {
  const fetchMock = answer(loaded(plans));
  render(PlannerPage, { props: { now } });

  expect(screen.getByRole('heading', { level: 1, name: 'Planner' })).toBeInTheDocument();
  expect(await screen.findByRole('heading', { level: 2, name: 'Your plans' })).toBeInTheDocument();
  expect(screen.getByText('3 plans')).toBeInTheDocument();
  expect(rows().map((row) => row.textContent?.replaceAll(/\s+/g, ' ').trim())).toEqual([
    'Today Back up the home lab',
    'Yesterday Untitled plan',
    '19 Oct Move the blog to a static site',
  ]);
  expect(within(rows()[0]).getByText('Today')).toHaveAttribute('datetime', '2026-10-31T00:20:00.123456789Z');
  expect(screen.queryByRole('link')).not.toBeInTheDocument();
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/planner/plans');
});

test('says when there is no plan yet, with no count', async () => {
  answer(loaded([]));
  render(PlannerPage, { props: { now } });

  expect(await screen.findByText('No plans yet.')).toBeInTheDocument();
  expect(screen.queryByText(/^\d+ plans?$/)).not.toBeInTheDocument();
});

test("says when Copilot isn't set up, with no section", async () => {
  answer(async () => new Response('Copilot is not set up\n', { status: 404 }));
  render(PlannerPage, { props: { now } });

  expect(await screen.findByText("Copilot isn't set up.")).toHaveClass('page-status');
  expect(screen.queryByRole('heading', { level: 2 })).not.toBeInTheDocument();
});

test("says when the plans couldn't load, in their section", async () => {
  answer(async () => new Response('Bad Gateway\n', { status: 502 }));
  render(PlannerPage, { props: { now } });

  expect(await screen.findByText("Couldn't load the plans.")).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 2, name: 'Your plans' })).toBeInTheDocument();
});

test('shows the head alone while the plans load', () => {
  answer(() => Promise.withResolvers<Response>().promise);
  const { container } = render(PlannerPage, { props: { now } });

  expect(screen.getByRole('heading', { level: 1, name: 'Planner' })).toBeInTheDocument();
  expect(container).toHaveTextContent(/^Planner Plans that Copilot works out with you\.$/);
});
