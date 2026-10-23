import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import html from '../../index.html?raw';
import { applyChoice, loadChoice, nextChoice, saveChoice, type ThemeChoice, themeColors } from './theme';

function metaContents() {
  return [...document.querySelectorAll<HTMLMetaElement>('meta[name="theme-color"]')].map((meta) => [
    meta.getAttribute('media'),
    meta.content,
  ]);
}

beforeEach(() => {
  // The page's own theme-color metas, without its scripts.
  const page = new DOMParser().parseFromString(html, 'text/html');
  document.head.replaceChildren(...page.querySelectorAll('meta[name="theme-color"]'));
});

afterEach(() => {
  localStorage.clear();
  document.documentElement.removeAttribute('data-theme');
  vi.restoreAllMocks();
});

test('cycles System, Light, Dark and back', () => {
  expect(nextChoice('system')).toBe('light');
  expect(nextChoice('light')).toBe('dark');
  expect(nextChoice('dark')).toBe('system');
});

test.each<ThemeChoice>(['light', 'dark', 'system'])('loads %s as it was saved', (choice) => {
  saveChoice(choice);

  expect(loadChoice()).toBe(choice);
});

test('leaves nothing in storage for System', () => {
  saveChoice('dark');
  saveChoice('system');

  expect(localStorage).toHaveLength(0);
});

test('loads an unknown value as System', () => {
  localStorage.setItem('theme', 'sepia');

  expect(loadChoice()).toBe('system');
});

test('loads System when storage throws', () => {
  vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
    throw new DOMException('denied', 'SecurityError');
  });

  expect(loadChoice()).toBe('system');
});

test.each<ThemeChoice>(['light', 'system'])('ignores a failing write of %s', (choice) => {
  for (const method of ['setItem', 'removeItem'] as const) {
    vi.spyOn(Storage.prototype, method).mockImplementation(() => {
      throw new DOMException('full', 'QuotaExceededError');
    });
  }

  expect(() => saveChoice(choice)).not.toThrow();
});

test("gives index.html's theme-color metas each scheme's colour", () => {
  expect(metaContents()).toEqual([
    ['(prefers-color-scheme: light)', themeColors.light],
    ['(prefers-color-scheme: dark)', themeColors.dark],
  ]);
});

test.each([
  { choice: 'light', theme: 'light', colors: [themeColors.light, themeColors.light] },
  { choice: 'dark', theme: 'dark', colors: [themeColors.dark, themeColors.dark] },
  { choice: 'system', theme: null, colors: [themeColors.light, themeColors.dark] },
] as const)('applies $choice', ({ choice, theme, colors }) => {
  applyChoice('dark');
  applyChoice(choice);

  expect(document.documentElement.getAttribute('data-theme')).toBe(theme);
  expect(metaContents().map(([, content]) => content)).toEqual(colors);
});
