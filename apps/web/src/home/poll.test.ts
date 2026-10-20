import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import type { Home } from './api';
import { type Feed, pollHome } from './poll';

const ada: Home = {
  name: 'Ada',
  items: [{ id: 'pulls/1', app: 'pulls', title: 'Nudge reviewers', source: 'Bitbucket', late: true }],
};
const grace: Home = { name: 'Grace', items: [] };
const start = new Date('2026-10-20T01:10:00Z');
const minute = 60_000;

let fetchMock = vi.fn<typeof fetch>();
let stop = () => {};

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(start);
  fetchMock = vi.fn<typeof fetch>(async () => Response.json(ada));
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  stop();
  vi.useRealTimers();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

function poll(): Feed[] {
  const feeds: Feed[] = [];
  stop = pollHome((feed) => feeds.push(feed));
  return feeds;
}

// Lets the loads in flight finish.
function settle(): Promise<unknown> {
  return vi.advanceTimersByTimeAsync(0);
}

function show(state: DocumentVisibilityState) {
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue(state);
  document.dispatchEvent(new Event('visibilitychange'));
}

function slowAnswer() {
  let answer: (response: Response) => void = () => {};
  fetchMock.mockImplementationOnce(() => new Promise((resolve) => (answer = resolve)));
  return (home: Home) => answer(Response.json(home));
}

test('loads the Home at once and reports it online', async () => {
  const feeds = poll();
  await settle();

  expect(fetchMock).toHaveBeenCalledWith('/api/home');
  expect(feeds).toEqual([{ connection: 'online', home: ada, asOf: start }]);
});

test('loads it again every minute', async () => {
  const feeds = poll();
  await settle();
  fetchMock.mockImplementation(async () => Response.json(grace));

  await vi.advanceTimersByTimeAsync(minute);

  expect(fetchMock).toHaveBeenCalledTimes(2);
  expect(feeds.at(-1)).toEqual({ connection: 'online', home: grace, asOf: new Date(start.getTime() + minute) });
});

test.each([
  { failure: 'an error status', fail: async () => new Response(null, { status: 503 }) },
  { failure: 'a network error', fail: () => Promise.reject(new TypeError('Failed to fetch')) },
])('keeps the last Home and its time, offline, after $failure', async ({ fail }) => {
  const feeds = poll();
  await settle();
  fetchMock.mockImplementation(fail);

  await vi.advanceTimersByTimeAsync(minute);

  expect(feeds.at(-1)).toEqual({ connection: 'offline', home: ada, asOf: start });
});

test('reports offline with no Home when the first load fails', async () => {
  fetchMock.mockRejectedValue(new TypeError('Failed to fetch'));

  const feeds = poll();
  await settle();

  expect(feeds).toEqual([{ connection: 'offline' }]);
});

test('skips the minute while the page is hidden, and loads when it shows again', async () => {
  poll();
  await settle();

  show('hidden');
  await vi.advanceTimersByTimeAsync(minute);
  expect(fetchMock).toHaveBeenCalledTimes(1);

  show('visible');
  expect(fetchMock).toHaveBeenCalledTimes(2);
});

test('ignores an answer that comes after a newer one', async () => {
  const answerFirst = slowAnswer();
  fetchMock.mockImplementation(async () => Response.json(grace));
  const feeds = poll();

  show('visible');
  await settle();
  answerFirst(ada);
  await settle();

  expect(feeds).toEqual([{ connection: 'online', home: grace, asOf: start }]);
});

test('reports nothing, loads nothing and listens to nothing once stopped', async () => {
  const answer = slowAnswer();
  const feeds = poll();

  stop();
  answer(ada);
  await settle();
  show('visible');
  await vi.advanceTimersByTimeAsync(minute);

  expect(feeds).toEqual([]);
  expect(fetchMock).toHaveBeenCalledTimes(1);
  expect(vi.getTimerCount()).toBe(0);
});
