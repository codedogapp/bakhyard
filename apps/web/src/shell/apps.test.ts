import { expect, test } from 'vitest';

import { appHref, apps } from './apps';

test('numbers the apps in the order of plan §7', () => {
  expect(apps.map((app) => [app.number, app.id, app.label, app.name])).toEqual([
    ['01', 'notes', 'Notes', 'Notes'],
    ['02', 'todos', 'Todos', 'Todos'],
    ['03', 'pulls', 'PRs', 'Pull requests'],
    ['04', 'time', 'Time', 'Time'],
    ['05', 'github', 'GitHub', 'GitHub'],
    ['06', 'games', 'Games', 'Games'],
    ['07', 'planner', 'Planner', 'Planner'],
  ]);
});

test('describes every app in a line', () => {
  for (const app of apps) {
    expect(app.description).not.toBe('');
  }
});

test("links to each app's page by its id", () => {
  expect(appHref('pulls')).toBe('/pulls');
});
