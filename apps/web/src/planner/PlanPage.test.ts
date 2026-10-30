import { act, fireEvent, render, screen, within } from '@testing-library/svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import type { Conversation } from './conversation';
import PlanPage from './PlanPage.svelte';

class FakeEvents {
  static readonly CLOSED = 2;
  static last: FakeEvents;
  listeners = new Map<string, (event: MessageEvent) => void>();
  readyState = 0;
  url: string;
  constructor(url: string) {
    this.url = url;
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
  // The browser gives up on the stream, as on a 404.
  fail() {
    this.readyState = FakeEvents.CLOSED;
    return act(() => this.listeners.get('error')?.(new MessageEvent('error')));
  }
}

const talk: Conversation = {
  messages: [
    { id: 'u1', from: 'user', text: 'Plan a trip.\nKeep it short.' },
    { id: 'a1', from: 'assistant', text: 'Where to?' },
    { id: 'u2', from: 'user', text: 'Rome' },
  ],
  writing: [],
  questions: [],
  plan: '',
  working: true,
  failure: '',
};
const city = { id: 'q1', text: 'Which city?', choices: ['Rome', 'Paris'], freeform: true };

const rows = () =>
  within(screen.getByRole('list'))
    .getAllByRole('listitem')
    .map((row) => row.textContent?.trim());

// The line that says whether Copilot works: the page's status, outside the reply form and the questions' rows.
const work = () => screen.getAllByRole('status').find((status) => status.closest('form, li') === null);

beforeEach(() => {
  vi.stubGlobal('EventSource', FakeEvents);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

test("shows the plan's conversation as it streams, and whether Copilot is working", async () => {
  render(PlanPage, { props: { id: 'p1' } });

  expect(screen.getByRole('heading', { level: 1, name: 'Plan' })).toBeInTheDocument();
  expect(FakeEvents.last.url).toBe('/api/planner/plans/p1/events');
  expect(screen.queryByRole('heading', { level: 2 })).not.toBeInTheDocument();

  await FakeEvents.last.emit('conversation', talk);
  expect(screen.getByRole('heading', { level: 2, name: 'The conversation' })).toBeInTheDocument();
  expect(rows()).toEqual(['You Plan a trip.\nKeep it short.', 'Copilot Where to?', 'You Rome']);
  expect(work()).toHaveTextContent('Copilot is working…');

  await FakeEvents.last.emit('delta', { id: 'a2', from: 'assistant', text: 'Rome it is. ' });
  await FakeEvents.last.emit('delta', { id: 'a2', from: 'assistant', text: 'Three days?' });
  expect(rows().at(-1)).toBe('Copilot Rome it is. Three days?');

  await FakeEvents.last.emit('status', { working: false, failure: 'Rate limited' });
  expect(rows()).toHaveLength(3);
  expect(work()).toHaveTextContent('Copilot stopped: Rate limited');
});

test('says when the plan has no message yet', async () => {
  render(PlanPage, { props: { id: 'p1' } });

  await FakeEvents.last.emit('conversation', { ...talk, messages: [], working: false });

  expect(screen.getByText('No messages yet.')).toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
  expect(work()?.textContent).toBe('');
  expect(screen.getByRole('textbox', { name: 'Reply' })).toBeInTheDocument();
});

test('offers a reply to the plan once its conversation has loaded', async () => {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 204 }));
  vi.stubGlobal('fetch', fetchMock);
  render(PlanPage, { props: { id: 'p1' } });
  expect(screen.queryByRole('textbox', { name: 'Reply' })).not.toBeInTheDocument();

  await FakeEvents.last.emit('conversation', talk);
  await fireEvent.input(screen.getByRole('textbox', { name: 'Reply' }), { target: { value: 'Make it a week' } });
  await fireEvent.click(screen.getByRole('button', { name: 'Send' }));

  expect(await screen.findByText('Sent')).toHaveRole('status');
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/planner/plans/p1/messages', expect.anything());
  expect(work()).toHaveTextContent('Copilot is working…');
});

test('shows the questions that wait after the messages, and hides the reply while one waits, keeping it', async () => {
  render(PlanPage, { props: { id: 'p1' } });
  await FakeEvents.last.emit('conversation', talk);
  await fireEvent.input(screen.getByRole('textbox', { name: 'Reply' }), { target: { value: 'Make it a week' } });

  await FakeEvents.last.emit('question', city);

  expect(screen.getAllByRole('listitem').at(-1)).toHaveTextContent(/^Copilot\s*Which city\?/);
  expect(screen.getByRole('button', { name: 'Rome' })).toBeInTheDocument();
  expect(work()).toHaveTextContent('Copilot waits for your answer.');
  expect(screen.queryByRole('textbox', { name: 'Reply' })).not.toBeInTheDocument();

  await FakeEvents.last.emit('answered', city);

  expect(screen.queryByText('Which city?')).not.toBeInTheDocument();
  expect(work()).toHaveTextContent('Copilot is working…');
  expect(screen.getByRole('textbox', { name: 'Reply' })).toHaveValue('Make it a week');
});

test('answers a question from its row, one that comes before any message too', async () => {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 204 }));
  vi.stubGlobal('fetch', fetchMock);
  render(PlanPage, { props: { id: 'p1' } });
  await FakeEvents.last.emit('conversation', { ...talk, messages: [], questions: [city] });
  expect(screen.queryByText('No messages yet.')).not.toBeInTheDocument();

  await fireEvent.click(screen.getByRole('button', { name: 'Rome' }));

  expect(await screen.findByText('Sent')).toHaveRole('status');
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/planner/plans/p1/questions/q1/answer', expect.anything());
});

test('keeps each answer typed with its question as another question goes', async () => {
  const days = { id: 'q2', text: 'How many days?', choices: [], freeform: true };
  render(PlanPage, { props: { id: 'p1' } });
  await FakeEvents.last.emit('conversation', { ...talk, questions: [city, days] });
  await fireEvent.input(screen.getByRole('textbox', { name: 'Answer to: How many days?' }), {
    target: { value: 'Three' },
  });

  await FakeEvents.last.emit('answered', city);

  expect(screen.getAllByRole('textbox')).toHaveLength(1);
  expect(screen.getByRole('textbox', { name: 'Answer to: How many days?' })).toHaveValue('Three');
});

test('leaves out a question that Copilot took back as it stopped, then and as it works again', async () => {
  render(PlanPage, { props: { id: 'p1' } });
  await FakeEvents.last.emit('conversation', { ...talk, questions: [city] });

  await FakeEvents.last.emit('status', { working: false, failure: '' });

  expect(screen.queryByText('Which city?')).not.toBeInTheDocument();
  expect(work()?.textContent).toBe('');
  expect(screen.getByRole('textbox', { name: 'Reply' })).toBeInTheDocument();

  await FakeEvents.last.emit('status', { working: true, failure: '' });

  expect(screen.queryByText('Which city?')).not.toBeInTheDocument();
  expect(work()).toHaveTextContent('Copilot is working…');
});

test("says when the plan doesn't load", async () => {
  render(PlanPage, { props: { id: 'p1' } });

  await FakeEvents.last.fail();

  expect(screen.getByText("Couldn't load the plan.")).toBeInTheDocument();
  expect(screen.queryByRole('heading', { level: 2 })).not.toBeInTheDocument();
  expect(screen.queryByRole('textbox', { name: 'Reply' })).not.toBeInTheDocument();
});

test("follows another plan from nothing when its ID changes, and the first plan's stream no more", async () => {
  const { rerender } = render(PlanPage, { props: { id: 'p1' } });
  const first = FakeEvents.last;
  await first.emit('conversation', talk);

  await rerender({ id: 'p2' });

  expect(first.readyState).toBe(FakeEvents.CLOSED);
  expect(FakeEvents.last.url).toBe('/api/planner/plans/p2/events');
  expect(screen.queryByRole('heading', { level: 2 })).not.toBeInTheDocument();
});

test('stops following the plan when it goes', () => {
  const { unmount } = render(PlanPage, { props: { id: 'p1' } });

  unmount();

  expect(FakeEvents.last.readyState).toBe(FakeEvents.CLOSED);
});
