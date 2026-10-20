import { expect, test } from 'vitest';

import { isCurrent, tabCurrent, tabItems } from './nav';

test.each([
  { href: '/', path: '/', want: true },
  { href: '/', path: '/notes', want: false },
  { href: '/notes', path: '/notes', want: true },
  { href: '/notes', path: '/notes/', want: true },
  { href: '/notes', path: '/notes/today', want: true },
  { href: '/notes', path: '/notes-archive', want: false },
  { href: '/notes', path: '/', want: false },
  { href: '/notes', path: '/time/notes', want: false },
])('isCurrent($href, $path) is $want', ({ href, path, want }) => {
  expect(isCurrent(href, path)).toBe(want);
});

test('gives Home, Notes, Todos and PRs a tab, then More', () => {
  expect(tabItems.map((item) => [item.label, item.href])).toEqual([
    ['Home', '/'],
    ['Notes', '/notes'],
    ['Todos', '/todos'],
    ['PRs', '/pulls'],
    ['More', '/more'],
  ]);
});

test.each([
  { path: '/', want: [['Home', 'page']] },
  { path: '/notes/today', want: [['Notes', 'page']] },
  { path: '/todos', want: [['Todos', 'page']] },
  { path: '/pulls', want: [['PRs', 'page']] },
  { path: '/more', want: [['More', 'page']] },
  { path: '/time', want: [['More', 'true']] },
  { path: '/github', want: [['More', 'true']] },
  { path: '/games/zelda', want: [['More', 'true']] },
  { path: '/planner', want: [['More', 'true']] },
  { path: '/more-ideas', want: [] },
  { path: '/elsewhere', want: [] },
])('marks the current tab at $path', ({ path, want }) => {
  const current = tabItems.map((item) => [item.label, tabCurrent(item.href, path)]).filter(([, how]) => how);
  expect(current).toEqual(want);
});
