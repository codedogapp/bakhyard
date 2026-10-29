import { expect, test } from 'vitest';

import { type Page, pageAt, pageTitle } from './page';

// The page at a path: its kind, with the app's id for an app's page and the note's path for a note's.
function pageId(path: string): string {
  const page = pageAt(path);
  switch (page.kind) {
    case 'app':
      return `app ${page.app.id}`;
    case 'note':
      return `note ${page.note.path}`;
    default:
      return page.kind;
  }
}

test.each([
  { path: '/', want: 'home' },
  { path: '/notes', want: 'notes' },
  { path: '/notes/', want: 'notes' },
  { path: '/notes/today', want: 'note today.md' },
  { path: '/notes/Daily/To%20do', want: 'note Daily/To do.md' },
  { path: '/notes//', want: 'missing' },
  { path: '/notes//today', want: 'missing' },
  { path: '/notes/Daily/', want: 'missing' },
  { path: '/notes/%E0%A4%A', want: 'missing' },
  { path: '/todos', want: 'todos' },
  { path: '/todos/', want: 'todos' },
  { path: '/todos/42', want: 'todos' },
  { path: '/todos-archive', want: 'missing' },
  { path: '/pulls', want: 'pulls' },
  { path: '/pulls/', want: 'pulls' },
  { path: '/pulls/42', want: 'pulls' },
  { path: '/pulls-archive', want: 'missing' },
  { path: '/time', want: 'time' },
  { path: '/time/', want: 'time' },
  { path: '/time/2026-10-21', want: 'time' },
  { path: '/time-archive', want: 'missing' },
  { path: '/github', want: 'app github' },
  { path: '/games/zelda', want: 'app games' },
  { path: '/planner', want: 'app planner' },
  { path: '/nudge', want: 'nudge' },
  { path: '/nudge/', want: 'nudge' },
  { path: '/nudge-archive', want: 'missing' },
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
  { page: pageAt('/notes/Daily/To%20do'), want: 'To do · Bakhyard' },
  { page: pageAt('/todos'), want: 'Todos · Bakhyard' },
  { page: pageAt('/time'), want: 'Time · Bakhyard' },
  { page: pageAt('/games'), want: 'Games · Bakhyard' },
  { page: pageAt('/pulls/42'), want: 'Pull requests · Bakhyard' },
  { page: pageAt('/nudge'), want: 'Nudge · Bakhyard' },
  { page: pageAt('/more'), want: 'More · Bakhyard' },
  { page: { kind: 'missing' } satisfies Page, want: 'Page not found · Bakhyard' },
])('titles the $page.kind page "$want"', ({ page, want }) => {
  expect(pageTitle(page)).toBe(want);
});
