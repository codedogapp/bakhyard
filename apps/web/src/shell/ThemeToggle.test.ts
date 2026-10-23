import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, expect, test } from 'vitest';

import ThemeToggle from './ThemeToggle.svelte';
import { saveChoice } from './theme';

afterEach(() => {
  localStorage.clear();
  document.documentElement.removeAttribute('data-theme');
});

test('names the saved choice', () => {
  saveChoice('dark');

  render(ThemeToggle);

  expect(screen.getByRole('button')).toHaveAccessibleName('Theme: Dark');
});

test('moves to the next choice on each click, and applies and saves it', async () => {
  render(ThemeToggle);
  const button = screen.getByRole('button', { name: 'Theme: System' });

  const seen = [];
  for (let click = 0; click < 3; click++) {
    await fireEvent.click(button);
    seen.push([
      button.getAttribute('aria-label'),
      document.documentElement.getAttribute('data-theme'),
      localStorage.getItem('theme'),
    ]);
  }

  expect(seen).toEqual([
    ['Theme: Light', 'light', 'light'],
    ['Theme: Dark', 'dark', 'dark'],
    ['Theme: System', null, null],
  ]);
});

test('shows a different icon for each choice', async () => {
  render(ThemeToggle);
  const button = screen.getByRole('button');

  const icons = new Set<string>();
  for (let click = 0; click < 3; click++) {
    icons.add(button.querySelector('svg')?.innerHTML ?? '');
    await fireEvent.click(button);
  }

  expect(icons.size).toBe(3);
});
