import { render, screen, within } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import TabBar from './TabBar.svelte';

test('links Home, Notes, Todos, PRs and More in order', () => {
  render(TabBar, { path: '/' });

  const links = within(screen.getByRole('navigation', { name: 'Apps' })).getAllByRole('link');
  expect(links.map((link) => [link.textContent, link.getAttribute('href')])).toEqual([
    ['Home', '/'],
    ['Notes', '/notes'],
    ['Todos', '/todos'],
    ['PRs', '/pulls'],
    ['More', '/more'],
  ]);
});

test.each([
  { path: '/', want: [['Home', 'page']] },
  { path: '/pulls/12', want: [['PRs', 'page']] },
  { path: '/games', want: [['More', 'true']] },
  { path: '/elsewhere', want: [] },
])('marks the current tab at $path', ({ path, want }) => {
  render(TabBar, { path });

  const current = screen
    .getAllByRole('link')
    .filter((link) => link.hasAttribute('aria-current'))
    .map((link) => [link.textContent, link.getAttribute('aria-current')]);
  expect(current).toEqual(want);
});
