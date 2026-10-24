import { expect, test } from 'vitest';

import { type Page, pageAt, pageTitle } from './page';

// The page at a path: its kind, and the app's id for an app's page.
function pageId(path: string): string {
  const page = pageAt(path);
  return page.kind === 'app' ? `app ${page.app.id}` : page.kind;
}

test.each([
  { path: '/', want: 'home' },
  { path: '/notes', want: 'notes' },
  { path: '/notes/', want: 'notes' },
  { path: '/notes/today', want: 'app notes' },
  { path: '/notes//', want: 'app notes' },
  { path: '/todos', want: 'app todos' },
  { path: '/pulls', want: 'app pulls' },
  { path: '/time', want: 'app time' },
  { path: '/github', want: 'app github' },
  { path: '/games/zelda', want: 'app games' },
  { path: '/planner', want: 'app planner' },
  { path: '/more', want: 'more' },
  { path: '/more/', want: 'more' },
  { path: '/elsewhere', want: 'missing' },
  { path: '/more-ideas', want: 'missing' },
  { path: '/notes-archive', want: 'missing' },
  { path: '/home', want: 'missing' },
])('shows $want at $path', ({ path, want }) => {
  expect(pageId(path)).toBe(want);
});

test.each([
  { page: pageAt('/'), want: 'Bakhyard' },
  { page: pageAt('/notes'), want: 'Notes · Bakhyard' },
  { page: pageAt('/time'), want: 'Time · Bakhyard' },
  { page: pageAt('/pulls/42'), want: 'Pull requests · Bakhyard' },
  { page: pageAt('/more'), want: 'More · Bakhyard' },
  { page: { kind: 'missing' } satisfies Page, want: 'Page not found · Bakhyard' },
])('titles the $page.kind page "$want"', ({ page, want }) => {
  expect(pageTitle(page)).toBe(want);
});
