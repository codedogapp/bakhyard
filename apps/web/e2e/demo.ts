import type { Home } from '../src/home/api';
import type { Todo } from '../src/todos/api';

/** A made-up user's Home at the tests' time (fixtures.ts): three items, one of them late. */
export const home: Home = {
  name: 'Alex',
  items: [
    {
      id: 'pulls/WEB-12',
      app: 'pulls',
      title: 'Nudge reviewers on WEB-12',
      source: 'Bitbucket',
      due: '2026-10-21T07:41:00+09:00',
      late: true,
    },
    {
      id: 'time/today',
      app: 'time',
      title: "Log today's hours",
      source: 'Jira',
      due: '2026-10-21T18:00:00+09:00',
      late: false,
    },
    { id: 'github/dotfiles-14', app: 'github', title: 'Review requested: dotfiles#14', source: 'GitHub', late: false },
  ],
  week: { loggedHours: 31.5, targetHours: 40 },
};

/** A made-up user's todos, newest first, one of each priority, so the page shows them in the same order. */
export const todos: Todo[] = [
  { id: 3, title: 'Write the release notes for 2.4', done: false, priority: 'high' },
  { id: 2, title: 'Book the car in for its winter tyres', done: false, priority: 'normal' },
  { id: 1, title: 'Renew the passport', done: true, priority: 'low' },
];
