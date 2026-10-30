import { expect, test } from 'vitest';

import { type Page, pageAt, pageTitle, planHref } from './page';

// The page at a path: its kind, with the app's id for an app's page, the note's path for a note's and the plan's ID for
// a plan's.
function pageId(path: string): string {
  const page = pageAt(path);
  switch (page.kind) {
    case 'app':
      return `app ${page.app.id}`;
    case 'note':
      return `note ${page.note.path}`;
    case 'plan':
      return `plan ${page.id}`;
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
  { path: '/github', want: 'github' },
  { path: '/github/', want: 'github' },
  { path: '/github/acme/web', want: 'github' },
  { path: '/github-archive', want: 'missing' },
  { path: '/games', want: 'games' },
  { path: '/games/', want: 'games' },
  { path: '/games/zelda', want: 'games' },
  { path: '/games-archive', want: 'missing' },
  { path: '/planner', want: 'planner' },
  { path: '/planner/', want: 'planner' },
  { path: '/planner/0b6c1f9e-2d4a-4c3b-9a8e-5f7d6c4b3a21', want: 'plan 0b6c1f9e-2d4a-4c3b-9a8e-5f7d6c4b3a21' },
  { path: '/planner/0b6c1f9e/', want: 'missing' },
  { path: '/planner/0b6c1f9e/messages', want: 'missing' },
  { path: '/planner//', want: 'missing' },
  { path: '/planner-archive', want: 'missing' },
  { path: '/nudge', want: 'nudge' },
  { path: '/nudge/', want: 'nudge' },
  { path: '/nudge-archive', want: 'missing' },
  { path: '/alerts', want: 'alerts' },
  { path: '/alerts/', want: 'alerts' },
  { path: '/alerts-archive', want: 'missing' },
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
  { id: '0b6c1f9e-2d4a-4c3b-9a8e-5f7d6c4b3a21', want: '/planner/0b6c1f9e-2d4a-4c3b-9a8e-5f7d6c4b3a21' },
  { id: 'a/b c?d#e', want: '/planner/a%2Fb%20c%3Fd%23e' },
])('gives the plan $id the page at $want, which shows it', ({ id, want }) => {
  expect(planHref(id)).toBe(want);
  expect(pageAt(want)).toEqual({ kind: 'plan', id: want.slice('/planner/'.length) });
});

test.each([
  { page: pageAt('/'), want: 'Bakhyard' },
  { page: pageAt('/notes'), want: 'Notes · Bakhyard' },
  { page: pageAt('/notes/Daily/To%20do'), want: 'To do · Bakhyard' },
  { page: pageAt('/todos'), want: 'Todos · Bakhyard' },
  { page: pageAt('/time'), want: 'Time · Bakhyard' },
  { page: pageAt('/github'), want: 'GitHub · Bakhyard' },
  { page: pageAt('/games'), want: 'Games · Bakhyard' },
  { page: pageAt('/planner'), want: 'Planner · Bakhyard' },
  { page: pageAt('/planner/0b6c1f9e-2d4a-4c3b-9a8e-5f7d6c4b3a21'), want: 'Plan · Bakhyard' },
  { page: pageAt('/pulls/42'), want: 'Pull requests · Bakhyard' },
  { page: pageAt('/nudge'), want: 'Nudge · Bakhyard' },
  { page: pageAt('/alerts'), want: 'Alerts · Bakhyard' },
  { page: pageAt('/more'), want: 'More · Bakhyard' },
  { page: { kind: 'missing' } satisfies Page, want: 'Page not found · Bakhyard' },
])('titles the $page.kind page "$want"', ({ page, want }) => {
  expect(pageTitle(page)).toBe(want);
});
