import { afterEach, expect, test, vi } from 'vitest';

import html from '../index.html?raw';
import source from '../public/theme.js?raw';
import { saveChoice } from './shell/theme';

// Runs public/theme.js as the browser runs it, from the start.
function runHeadScript() {
  new Function(source)();
}

afterEach(() => {
  localStorage.clear();
  document.documentElement.removeAttribute('data-theme');
  vi.restoreAllMocks();
});

test("loads /theme.js in the head as a classic script, so it runs before the body's first paint", () => {
  const page = new DOMParser().parseFromString(html, 'text/html');
  const script = page.querySelector('script[src="/theme.js"]');

  expect(script?.parentElement).toBe(page.head);
  // No type, async or defer: the parser waits for it.
  expect(script?.getAttributeNames()).toEqual(['src']);
});

test.each(['light', 'dark'] as const)('sets the saved %s scheme', (choice) => {
  saveChoice(choice);

  runHeadScript();

  expect(document.documentElement).toHaveAttribute('data-theme', choice);
});

test.each([
  { name: 'nothing saved', stored: null },
  { name: 'an unknown value', stored: 'sepia' },
])('leaves the scheme to the OS with $name', ({ stored }) => {
  if (stored) {
    localStorage.setItem('theme', stored);
  }

  runHeadScript();

  expect(document.documentElement).not.toHaveAttribute('data-theme');
});

test('leaves the scheme to the OS when storage throws', () => {
  vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
    throw new DOMException('denied', 'SecurityError');
  });

  expect(runHeadScript).not.toThrow();
  expect(document.documentElement).not.toHaveAttribute('data-theme');
});
