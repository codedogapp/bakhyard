import { expect, test } from 'vitest';

import { isCurrent } from './nav';

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
