import { render, screen, within } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import Header from './Header.svelte';

test('links the wordmark to Home', () => {
  render(Header, { path: '/' });

  const banner = screen.getByRole('banner');
  expect(within(banner).getByRole('link', { name: 'bakhyard' })).toHaveAttribute('href', '/');
});

test('links every app in order', () => {
  render(Header, { path: '/' });

  const links = within(screen.getByRole('navigation', { name: 'Apps' })).getAllByRole('link');
  expect(links.map((link) => [link.textContent, link.getAttribute('href')])).toEqual([
    ['Home', '/'],
    ['Notes', '/notes'],
    ['Todos', '/todos'],
    ['PRs', '/pulls'],
    ['Time', '/time'],
    ['GitHub', '/github'],
    ['Games', '/games'],
    ['Planner', '/planner'],
  ]);
});

test.each([
  { path: '/', want: ['Home'] },
  { path: '/notes/today', want: ['Notes'] },
  { path: '/elsewhere', want: [] },
])('marks $want as the current page at $path', ({ path, want }) => {
  render(Header, { path });

  const nav = screen.getByRole('navigation', { name: 'Apps' });
  const current = within(nav)
    .getAllByRole('link')
    .filter((link) => link.getAttribute('aria-current') === 'page');
  expect(current.map((link) => link.textContent)).toEqual(want);
});
