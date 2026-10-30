import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import type { Conversation } from './conversation';
import { type Chat, followPlan, reopenMs } from './follow';

const connecting = 0;
const closed = 2;

class FakeEvents {
  static readonly CLOSED = closed;
  static all: FakeEvents[] = [];
  listeners = new Map<string, (event: MessageEvent) => void>();
  readyState = connecting;
  url: string;
  constructor(url: string) {
    this.url = url;
    FakeEvents.all.push(this);
  }
  addEventListener(name: string, listener: (event: MessageEvent) => void) {
    this.listeners.set(name, listener);
  }
  close() {
    this.readyState = closed;
  }
  emit(name: string, data: unknown) {
    this.listeners.get(name)?.(new MessageEvent(name, { data: JSON.stringify(data) }));
  }
  // The browser connects again by itself while the state is connecting, and has given up once it's closed.
  fail(state: number) {
    this.readyState = state;
    this.listeners.get('error')?.(new MessageEvent('error'));
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

let shown: Chat[] = [];
let stop = () => {};
const last = () => FakeEvents.all.at(-1) as FakeEvents;

beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal('EventSource', FakeEvents);
  FakeEvents.all = [];
  shown = [];
  stop = followPlan('p1', (chat) => shown.push(chat));
});

afterEach(() => {
  stop();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

test("follows the plan's stream, and shows its conversation, then each change", () => {
  expect(last().url).toBe('/api/planner/plans/p1/events');

  const where = { id: 'a1', from: 'assistant', text: 'Where' } as const;
  const city = { id: 'q1', text: 'Which city?', choices: [], freeform: true };

  last().emit('conversation', talk);
  last().emit('delta', where);
  last().emit('question', city);
  last().emit('answered', city);
  last().emit('plan', { plan: '# Trip' });
  last().emit('status', { working: false, failure: 'Rate limited' });

  expect(shown).toEqual([
    { kind: 'loaded', talk },
    { kind: 'loaded', talk: { ...talk, writing: [where] } },
    { kind: 'loaded', talk: { ...talk, writing: [where], questions: [city] } },
    { kind: 'loaded', talk: { ...talk, writing: [where] } },
    { kind: 'loaded', talk: { ...talk, writing: [where], plan: '# Trip' } },
    { kind: 'loaded', talk: { ...talk, plan: '# Trip', working: false, failure: 'Rate limited' } },
  ]);
});

test("says that the plan didn't load when its stream doesn't open, and tries no more", () => {
  last().fail(closed);
  vi.advanceTimersByTime(reopenMs);

  expect(shown).toEqual([{ kind: 'failed' }]);
  expect(FakeEvents.all).toHaveLength(1);
});

test('leaves a broken stream to the browser while it connects again', () => {
  last().emit('conversation', talk);

  last().fail(connecting);
  vi.advanceTimersByTime(reopenMs);

  expect(shown).toEqual([{ kind: 'loaded', talk }]);
  expect(FakeEvents.all).toHaveLength(1);
});

test('opens the stream again a while after the browser gives up, and keeps the conversation till the new one', () => {
  last().emit('conversation', talk);

  last().fail(closed);
  vi.advanceTimersByTime(reopenMs - 1);
  expect(FakeEvents.all).toHaveLength(1);
  vi.advanceTimersByTime(1);
  expect(FakeEvents.all).toHaveLength(2);
  expect(last().url).toBe('/api/planner/plans/p1/events');
  last().emit('conversation', { ...talk, working: false });

  expect(shown).toEqual([
    { kind: 'loaded', talk },
    { kind: 'loaded', talk: { ...talk, working: false } },
  ]);
});

test('drops a reopen that waits when it stops', () => {
  last().emit('conversation', talk);
  last().fail(closed);

  stop();
  vi.advanceTimersByTime(reopenMs);

  expect(FakeEvents.all).toHaveLength(1);
});

test('closes the stream it follows when it stops, a reopened one too', () => {
  last().emit('conversation', talk);
  last().fail(closed);
  vi.advanceTimersByTime(reopenMs);

  stop();

  expect(FakeEvents.all).toHaveLength(2);
  expect(last().readyState).toBe(closed);
});
