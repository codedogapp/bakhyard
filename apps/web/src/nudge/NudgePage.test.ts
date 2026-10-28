import { act, render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import NudgePage from './NudgePage.svelte';
import type { State } from './nudge';

class FakeEvents {
  static last: FakeEvents;
  listeners = new Map<string, () => void>();
  constructor() {
    FakeEvents.last = this;
  }
  addEventListener(name: string, listener: () => void) {
    this.listeners.set(name, listener);
  }
  close() {}
}

const empty: State = { questions: [], sessions: [] };
const asked: State = {
  questions: [
    { id: '1', project: 'atlas', text: 'Merge?', details: '', options: ['Yes'], asked_at: '2026-10-21T00:41:00Z' },
  ],
  sessions: [{ id: 's', project: 'atlas', note: 'Reading', noted_at: '2026-10-21T00:40:00Z' }],
};

let current: State | number = empty;

beforeEach(() => {
  current = empty;
  vi.stubGlobal('EventSource', FakeEvents);
  vi.stubGlobal(
    'fetch',
    vi.fn<typeof fetch>(async () =>
      typeof current === 'number' ? new Response('', { status: current }) : Response.json(current)
    )
  );
});

afterEach(() => {
  vi.unstubAllGlobals();
});

test('loads the questions and the sessions, under its head', async () => {
  current = asked;
  render(NudgePage);

  expect(screen.getByRole('heading', { level: 1, name: 'Nudge' })).toBeInTheDocument();
  expect(await screen.findByRole('heading', { level: 3, name: 'Merge?' })).toBeInTheDocument();
  expect(screen.getByText('Reading')).toBeInTheDocument();
  // jsdom has no push manager, so notifications can't be turned on here.
  expect(await screen.findByText('Notifications need the Home Screen app')).toBeInTheDocument();
});

test('shows a question that arrives while it is open, without a reload', async () => {
  render(NudgePage);
  expect(await screen.findByText('No agent is waiting for you.')).toBeInTheDocument();

  current = asked;
  await act(() => FakeEvents.last.listeners.get('changed')?.());

  expect(await screen.findByRole('heading', { level: 3, name: 'Merge?' })).toBeInTheDocument();
});

test('keeps what it shows when a later load fails', async () => {
  current = asked;
  render(NudgePage);
  await screen.findByRole('heading', { level: 3, name: 'Merge?' });

  current = 502;
  await act(() => FakeEvents.last.listeners.get('changed')?.());

  expect(screen.getByRole('heading', { level: 3, name: 'Merge?' })).toBeInTheDocument();
  expect(screen.queryByText("Couldn't load the questions.")).not.toBeInTheDocument();
});

test("says when it couldn't load", async () => {
  current = 502;
  render(NudgePage);

  expect(await screen.findByText("Couldn't load the questions.")).toBeInTheDocument();
});
