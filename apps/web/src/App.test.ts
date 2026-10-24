import { act, fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import App from './App.svelte';
import type { Home } from './home/api';

const home: Home = {
  name: 'Ada',
  items: [
    {
      id: 'pulls/1',
      app: 'pulls',
      title: 'Nudge reviewers',
      source: 'Bitbucket',
      due: '2026-10-20T00:59:59Z',
      late: true,
    },
    // 30 seconds after 11:59:59 in Tokyo, the tests' time zone: "in 1 minute" then, and "now" at 12:00.
    { id: 'time/1', app: 'time', title: "Log today's hours", source: 'Jira', due: '2026-10-20T03:00:29Z', late: false },
  ],
  week: { loggedHours: 31.5, targetHours: 40 },
};

// Fires the navigate event the browser fires for a click on a link to path; returns its handler if intercepted.
function navigate(navigation: EventTarget, path: string): NavigationInterceptHandler | undefined {
  let handler: NavigationInterceptHandler | undefined;
  navigation.dispatchEvent(
    Object.assign(new Event('navigate'), {
      canIntercept: true,
      hashChange: false,
      downloadRequest: null,
      formData: null,
      navigationType: 'push',
      destination: { url: new URL(path, location.href).href },
      intercept: (options?: NavigationInterceptOptions) => {
        handler = options?.handler;
      },
    })
  );
  return handler;
}

// Navigates to path as the browser does, and waits for the page to show.
async function go(navigation: EventTarget, path: string) {
  const handler = navigate(navigation, path);
  expect(handler).toBeDefined();
  await act(async () => {
    await handler?.();
  });
}

const notes = { files: ['Work/Plan.md', 'Inbox.md'] };

beforeEach(() => {
  vi.stubGlobal(
    'fetch',
    vi.fn<typeof fetch>(async (url) => Response.json(url === '/api/notes' ? notes : home))
  );
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  history.replaceState(null, '', '/');
});

test('shows the frame and the apps, then the hero, the feed, the counts, the week and the status once the Home loads', async () => {
  render(App);

  expect(screen.getByText(/ · Tokyo$/)).toBeInTheDocument();
  expect(screen.getByRole('banner')).toBeInTheDocument();
  // The header's nav and the tab bar: the CSS shows one of them, by the screen's width.
  expect(screen.getAllByRole('link', { name: 'Home', current: 'page' })).toHaveLength(2);
  expect(screen.queryByRole('heading', { level: 2, name: 'Needs you today' })).not.toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 2, name: 'Your apps' })).toBeInTheDocument();
  expect(screen.queryByText(/^items?$/)).not.toBeInTheDocument();
  expect(screen.queryByRole('heading', { level: 2, name: 'This week' })).not.toBeInTheDocument();
  expect(await screen.findByRole('heading', { level: 1, name: /^Good \w+, Ada\.$/ })).toBeInTheDocument();
  expect(screen.getByText('2 things need you today. 1 is late.')).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 2, name: 'Needs you today' })).toBeInTheDocument();
  expect(screen.getByText('Nudge reviewers')).toBeInTheDocument();
  expect(screen.getByText("Log today's hours")).toBeInTheDocument();
  expect(screen.getAllByText('item')).toHaveLength(2);
  expect(screen.getByRole('heading', { level: 2, name: 'This week' })).toBeInTheDocument();
  expect(screen.getByText('31h 30m / 40h')).toBeInTheDocument();
  expect(screen.getByRole('status')).toHaveTextContent('Online');
  expect(document.title).toBe('Bakhyard');
});

test('shows the page at the path it opens on', () => {
  history.replaceState(null, '', '/time');
  render(App);

  expect(screen.getByRole('heading', { level: 1, name: 'Time' })).toBeInTheDocument();
  expect(screen.getByText('Coming soon')).toBeInTheDocument();
  expect(document.title).toBe('Time · Bakhyard');
  expect(screen.getByRole('link', { name: 'Time', current: 'page' })).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'More', current: true })).toBeInTheDocument();
});

test('shows each page in place as the browser navigates, with the Home it has already loaded', async () => {
  const navigation = new EventTarget();
  vi.stubGlobal('navigation', navigation);
  render(App);
  await screen.findByRole('heading', { level: 1, name: /^Good \w+, Ada\.$/ });

  await go(navigation, '/more');

  expect(screen.getByRole('heading', { level: 1, name: 'More' })).toBeInTheDocument();
  expect(screen.getAllByRole('heading', { level: 2 })).toHaveLength(4);
  expect(screen.getByRole('heading', { level: 2, name: 'Time' }).closest('a')).toHaveTextContent('1 item');
  expect(document.title).toBe('More · Bakhyard');
  expect(screen.getByRole('link', { name: 'More', current: 'page' })).toBeInTheDocument();

  await go(navigation, '/elsewhere');

  expect(screen.getByRole('heading', { level: 1, name: 'Page not found' })).toBeInTheDocument();
  expect(screen.getByText("There's no page at /elsewhere.")).toBeInTheDocument();
  expect(document.title).toBe('Page not found · Bakhyard');
  expect(screen.queryByRole('link', { current: 'page' })).not.toBeInTheDocument();

  await go(navigation, '/');

  expect(screen.getByRole('heading', { level: 1, name: /^Good \w+, Ada\.$/ })).toBeInTheDocument();
  expect(document.title).toBe('Bakhyard');
  expect(fetch).toHaveBeenCalledOnce();
});

test('keeps the Notes page as the user left it while the app runs, with the list it has already loaded', async () => {
  const navigation = new EventTarget();
  vi.stubGlobal('navigation', navigation);
  history.replaceState(null, '', '/notes');
  render(App);

  expect(await screen.findByText('2 notes in the vault.')).toBeInTheDocument();
  expect(document.title).toBe('Notes · Bakhyard');
  expect(screen.getAllByRole('link', { name: 'Notes', current: 'page' })).toHaveLength(2);
  const work = screen.getByText('Work').closest('details')!;
  const opened = new Promise((resolve) => work.addEventListener('toggle', resolve, { once: true }));
  await fireEvent.click(screen.getByText('Work'));
  await opened;

  await go(navigation, '/notes/Work/Plan');

  expect(screen.getByRole('heading', { level: 1, name: 'Notes' })).toBeInTheDocument();
  expect(screen.getByText('Coming soon')).toBeInTheDocument();

  await go(navigation, '/notes');

  expect(screen.getByText('Work').closest('details')).toHaveProperty('open', true);
  expect(screen.getByRole('link', { name: 'Plan' })).toHaveAttribute('href', '/notes/Work/Plan');
  expect(vi.mocked(fetch).mock.calls.filter(([url]) => url === '/api/notes')).toHaveLength(1);
});

test('leaves navigations to the browser once it goes away', () => {
  const navigation = new EventTarget();
  vi.stubGlobal('navigation', navigation);
  const { unmount } = render(App);
  expect(navigate(navigation, '/more')).toBeDefined();

  unmount();

  expect(navigate(navigation, '/more')).toBeUndefined();
});

test('moves the clock, the greeting and the due times on together', async () => {
  vi.useFakeTimers();
  // 11:59:59 on Tuesday 20 October in Tokyo.
  vi.setSystemTime(new Date('2026-10-20T02:59:59Z'));
  render(App);
  await act(() => vi.advanceTimersByTimeAsync(0));

  expect(screen.getByText('11:59 · Tokyo')).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 1, name: 'Good morning, Ada.' })).toBeInTheDocument();
  expect(screen.getByText('in 1 minute')).toBeInTheDocument();

  await act(() => vi.advanceTimersByTimeAsync(1000));

  expect(screen.getByText('12:00 · Tokyo')).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 1, name: 'Good afternoon, Ada.' })).toBeInTheDocument();
  expect(screen.getByText('now')).toBeInTheDocument();
});

test('shows only the top bar and the sign-in screen while signed out, and the Home once signed in again', async () => {
  vi.useFakeTimers();
  vi.mocked(fetch).mockImplementation(async () => new Response(null, { status: 401 }));
  render(App);
  await act(() => vi.advanceTimersByTimeAsync(0));

  expect(screen.getByRole('status')).toHaveTextContent('Signed out');
  expect(screen.getByRole('heading', { level: 1, name: 'Signed out' })).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Sign in' })).toHaveAttribute('href', '/auth/google/login');
  expect(screen.queryByRole('banner')).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Sign out' })).not.toBeInTheDocument();
  expect(screen.queryByRole('navigation')).not.toBeInTheDocument();
  expect(document.title).toBe('Signed out · Bakhyard');

  vi.mocked(fetch).mockImplementation(async () => Response.json(home));
  await act(() => vi.advanceTimersByTimeAsync(60_000));

  expect(screen.getByRole('heading', { level: 1, name: /^Good \w+, Ada\.$/ })).toBeInTheDocument();
  expect(screen.getByRole('banner')).toBeInTheDocument();
  expect(document.title).toBe('Bakhyard');
});

test('stops its clock and its polling when it goes away', async () => {
  vi.useFakeTimers();
  const { unmount } = render(App);
  await act(() => vi.advanceTimersByTimeAsync(0));
  expect(vi.getTimerCount()).toBe(2);

  unmount();

  expect(vi.getTimerCount()).toBe(0);
});
