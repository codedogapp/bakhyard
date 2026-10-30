import { afterEach, expect, test, vi } from 'vitest';

import { followNavigation, openPage } from './navigation';

type Intercept = (options?: NavigationInterceptOptions) => void;

// A navigate event as the browser fires it for a click on a same-origin link, with any of its fields changed.
function navigate(target: EventTarget, fields: Partial<NavigateEvent> = {}) {
  const intercept = vi.fn<Intercept>();
  target.dispatchEvent(
    Object.assign(new Event('navigate'), {
      canIntercept: true,
      hashChange: false,
      downloadRequest: null,
      formData: null,
      navigationType: 'push',
      destination: { url: 'http://localhost:3000/time/report?week=2#today' },
      intercept,
      ...fields,
    })
  );
  return intercept;
}

function followed() {
  const navigation = new EventTarget();
  vi.stubGlobal('navigation', navigation);
  const settled = Promise.resolve();
  const onPath = vi.fn((_path: string) => settled);
  const stop = followNavigation(onPath);
  return { navigation, settled, onPath, stop };
}

afterEach(() => {
  vi.unstubAllGlobals();
  history.replaceState(null, '', '/');
});

test('leaves every navigation to the browser without the Navigation API', () => {
  expect('navigation' in globalThis).toBe(false);
  const onPath = vi.fn(() => Promise.resolve());

  followNavigation(onPath)();

  expect(onPath).not.toHaveBeenCalled();
});

test("shows the destination's path in place, and the browser waits for it", () => {
  const { navigation, settled, onPath } = followed();

  const intercept = navigate(navigation);

  expect(intercept).toHaveBeenCalledOnce();
  expect(onPath).not.toHaveBeenCalled();
  const handler = intercept.mock.calls[0]?.[0]?.handler;
  expect(handler?.()).toBe(settled);
  expect(onPath).toHaveBeenCalledWith('/time/report');
});

test.each([
  { name: 'back and forward', fields: { navigationType: 'traverse' } },
  { name: 'a replacement', fields: { navigationType: 'replace' } },
] as const)('shows the page in place on $name', ({ fields }) => {
  const { navigation } = followed();

  expect(navigate(navigation, fields)).toHaveBeenCalledOnce();
});

test.each([
  { name: "a navigation it can't intercept", fields: { canIntercept: false } },
  { name: 'a jump to an anchor', fields: { hashChange: true } },
  { name: 'a download', fields: { downloadRequest: 'report.csv' } },
  { name: "a form's submission", fields: { formData: new FormData() } },
  { name: 'a reload', fields: { navigationType: 'reload' } },
] as const)('leaves $name to the browser', ({ fields }) => {
  const { navigation } = followed();

  expect(navigate(navigation, fields)).not.toHaveBeenCalled();
});

test("leaves the gateway's own paths, where signing in starts, to the browser", () => {
  const { navigation } = followed();
  const destination = { url: 'http://localhost:3000/auth/google/login' } as NavigationDestination;

  expect(navigate(navigation, { destination })).not.toHaveBeenCalled();
});

test('leaves navigations to the browser once stopped', () => {
  const { navigation, stop } = followed();

  stop();

  expect(navigate(navigation)).not.toHaveBeenCalled();
});

test('opens the page at a path through the location, as a link would', () => {
  // jsdom navigates only within the document, so the test opens an anchor on the page it's at.
  openPage('/#plans');

  expect(location.href).toBe('http://localhost:3000/#plans');
});
