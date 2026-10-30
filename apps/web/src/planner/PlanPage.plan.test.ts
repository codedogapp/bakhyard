import { act, render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import type { Conversation } from './conversation';
import PlanPage from './PlanPage.svelte';

class FakeEvents {
  static readonly CLOSED = 2;
  static last: FakeEvents;
  listeners = new Map<string, (event: MessageEvent) => void>();
  readyState = 0;
  constructor() {
    FakeEvents.last = this;
  }
  addEventListener(name: string, listener: (event: MessageEvent) => void) {
    this.listeners.set(name, listener);
  }
  close() {
    this.readyState = FakeEvents.CLOSED;
  }
  emit(name: string, data: unknown) {
    return act(() => this.listeners.get(name)?.(new MessageEvent(name, { data: JSON.stringify(data) })));
  }
}

const talk: Conversation = {
  messages: [{ id: 'u1', from: 'user', text: 'Plan a trip' }],
  writing: [],
  questions: [],
  plan: '',
  working: true,
  failure: '',
};
const trip = '# Trip\n\n- [ ] Book the train';

// The page's headings, in the order they come, once the plan's renderer has loaded.
async function headings(): Promise<string[]> {
  await screen.findByRole('article');
  return screen.getAllByRole('heading').map((heading) => `${heading.tagName} ${heading.textContent}`);
}

beforeEach(() => {
  vi.stubGlobal('EventSource', FakeEvents);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

test('shows no plan till Copilot brings one, then the plan, rendered, above the conversation', async () => {
  render(PlanPage, { props: { id: 'p1' } });
  await FakeEvents.last.emit('conversation', talk);
  // A plan's renderer loads apart from the page, so it has to have had the time to.
  await vi.dynamicImportSettled();
  expect(screen.queryByRole('article')).not.toBeInTheDocument();

  await FakeEvents.last.emit('plan', { plan: trip });

  expect(await headings()).toEqual(['H1 Plan', 'H2 Trip', 'H2 The conversation']);
  expect(screen.getByRole('checkbox')).not.toBeChecked();
  expect(screen.getByRole('article')).toHaveTextContent(/^Trip\s*Book the train$/);
});

test('shows the plan that came before the page opened', async () => {
  render(PlanPage, { props: { id: 'p1' } });

  await FakeEvents.last.emit('conversation', { ...talk, plan: trip, working: false });

  expect(await headings()).toEqual(['H1 Plan', 'H2 Trip', 'H2 The conversation']);
});

test('shows a new plan in place of the old one, and keeps it once Copilot stops', async () => {
  render(PlanPage, { props: { id: 'p1' } });
  await FakeEvents.last.emit('conversation', { ...talk, plan: trip });
  await screen.findByRole('article');

  await FakeEvents.last.emit('plan', { plan: '# Rome trip' });
  await FakeEvents.last.emit('status', { working: false, failure: '' });

  expect(await headings()).toEqual(['H1 Plan', 'H2 Rome trip', 'H2 The conversation']);
});

test('renders the plan once, not again with each piece of a message, which would slow a phone', async () => {
  const renderPlan = vi.fn((markdown: string) => `<p>${markdown}</p>`);
  vi.doMock('./render', () => ({ renderPlan }));
  try {
    render(PlanPage, { props: { id: 'p1' } });
    await FakeEvents.last.emit('conversation', { ...talk, plan: 'Trip' });
    expect(await screen.findByRole('article')).toHaveTextContent('Trip');

    await FakeEvents.last.emit('delta', { id: 'a1', from: 'assistant', text: 'Rome' });
    await FakeEvents.last.emit('delta', { id: 'a1', from: 'assistant', text: ' it is.' });

    expect(renderPlan).toHaveBeenCalledOnce();
  } finally {
    vi.doUnmock('./render');
  }
});

test("says so when the plan can't show, and shows the conversation still", async () => {
  vi.doMock('./render', () => {
    throw new Error('Failed to fetch dynamically imported module');
  });
  try {
    render(PlanPage, { props: { id: 'p1' } });
    await FakeEvents.last.emit('conversation', { ...talk, plan: trip });

    expect(await screen.findByText("Couldn't show the plan.")).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 2, name: 'The conversation' })).toBeInTheDocument();
  } finally {
    vi.doUnmock('./render');
  }
});
