import type { Home } from '../src/home/api';
import type { Todo } from '../src/todos/api';
import type { Snapshot } from '../src/todos/tickets';

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

/** A made-up user's open Jira tickets, as the todos service found them three minutes before the tests' time. */
export const tickets: Snapshot = {
  tickets: [
    {
      key: 'WEB-12',
      summary: 'Show the order history on the account page',
      status: 'In Review',
      category: 'in-progress',
      url: 'https://jira.example.com/browse/WEB-12',
    },
    {
      key: 'WEB-9',
      summary: 'Fix the date picker in Safari',
      status: 'In Progress',
      category: 'in-progress',
      url: 'https://jira.example.com/browse/WEB-9',
    },
    {
      key: 'OPS-31',
      summary: 'Rotate the staging certificates',
      status: 'To Do',
      category: 'to-do',
      url: 'https://jira.example.com/browse/OPS-31',
    },
  ],
  searched: '2026-10-21T00:38:00Z',
};
