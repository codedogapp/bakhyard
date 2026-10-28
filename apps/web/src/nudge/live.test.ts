import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import { fallbackMs, stayCurrent } from './live';

class FakeEvents {
  static last: FakeEvents;
  listeners = new Map<string, () => void>();
  closed = false;
  url: string;
  constructor(url: string) {
    this.url = url;
    FakeEvents.last = this;
  }
  addEventListener(name: string, listener: () => void) {
    this.listeners.set(name, listener);
  }
  close() {
    this.closed = true;
  }
  emit(name: string) {
    this.listeners.get(name)?.();
  }
}

const worker = new EventTarget();
let refresh = vi.fn();
let stop = () => {};

beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal('EventSource', FakeEvents);
  vi.stubGlobal('navigator', { serviceWorker: worker });
  refresh = vi.fn();
  stop = stayCurrent(refresh);
});

afterEach(() => {
  stop();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

function show(state: 'visible' | 'hidden') {
  Object.defineProperty(document, 'visibilityState', { value: state, configurable: true });
}

test('listens to the service on its event stream', () => {
  expect(FakeEvents.last.url).toBe('/api/nudge/events');

  FakeEvents.last.emit('changed');

  expect(refresh).toHaveBeenCalledTimes(1);
});

test('refreshes when the stream opens again, but not when it first opens', () => {
  FakeEvents.last.emit('open');
  expect(refresh).not.toHaveBeenCalled();

  FakeEvents.last.emit('open');
  expect(refresh).toHaveBeenCalledTimes(1);
});

test('refreshes when the service worker tells of a push', () => {
  worker.dispatchEvent(new MessageEvent('message', { data: { kind: 'push' } }));

  expect(refresh).toHaveBeenCalledTimes(1);
});

test('refreshes when the page comes back on screen, not when it goes away', () => {
  show('hidden');
  document.dispatchEvent(new Event('visibilitychange'));
  expect(refresh).not.toHaveBeenCalled();

  show('visible');
  document.dispatchEvent(new Event('visibilitychange'));
  expect(refresh).toHaveBeenCalledTimes(1);
});

test('refreshes every minute while the page is on screen only', () => {
  show('visible');
  vi.advanceTimersByTime(fallbackMs);
  expect(refresh).toHaveBeenCalledTimes(1);

  show('hidden');
  vi.advanceTimersByTime(fallbackMs);
  expect(refresh).toHaveBeenCalledTimes(1);
});

test('stops listening', () => {
  stop();
  show('visible');

  document.dispatchEvent(new Event('visibilitychange'));
  worker.dispatchEvent(new MessageEvent('message'));
  vi.advanceTimersByTime(fallbackMs);

  expect(FakeEvents.last.closed).toBe(true);
  expect(refresh).not.toHaveBeenCalled();
});

test('works in a browser without a service worker', () => {
  stop();
  vi.stubGlobal('navigator', {});

  stop = stayCurrent(refresh);
  stop();

  expect(refresh).not.toHaveBeenCalled();
});
