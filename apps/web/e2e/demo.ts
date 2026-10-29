import type { Home } from '../src/home/api';
import type { Snapshot } from '../src/jira/tickets';
import type { State } from '../src/nudge/nudge';
import type { Snapshot as Pulls } from '../src/pulls/pulls';
import type { Report } from '../src/time/report';
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

/** A made-up user's open PRs on Bitbucket, the latest updated first, as the pulls service found them three minutes before the tests' time. */
export const pulls: Pulls = {
  pulls: [
    {
      id: 128,
      title: 'WEB-12 Show the order history on the account page',
      repo: 'acme/web',
      url: 'https://bitbucket.org/acme/web/pull-requests/128',
      draft: false,
      reviewers: [
        { name: 'Mia Chen', verdict: 'approved' },
        { name: 'Leo Hart', verdict: 'pending' },
      ],
    },
    {
      id: 41,
      title: 'Retry the payment webhook after a timeout',
      repo: 'acme/payments',
      url: 'https://bitbucket.org/acme/payments/pull-requests/41',
      draft: true,
    },
    {
      id: 126,
      title: 'WEB-9 Fix the date picker in Safari',
      repo: 'acme/web',
      url: 'https://bitbucket.org/acme/web/pull-requests/126',
      draft: false,
      reviewers: [
        { name: 'Tom Berg', verdict: 'pending' },
        { name: 'Noor Aziz', verdict: 'changes_requested' },
        { name: 'Mia Chen', verdict: 'approved' },
        { name: 'Leo Hart', verdict: 'pending' },
      ],
    },
  ],
  searched: '2026-10-21T00:38:00Z',
};

/**
 * The open PRs that wait for a made-up user's review, the latest updated first, as the pulls service found them
 * eleven minutes before the tests' time.
 */
export const reviewRequests: Pulls = {
  pulls: [
    {
      id: 131,
      title: 'WEB-14 Cache the product images at the edge',
      repo: 'acme/web',
      url: 'https://bitbucket.org/acme/web/pull-requests/131',
      draft: false,
    },
    {
      id: 87,
      title: 'Add the monthly invoice export',
      repo: 'acme/billing',
      url: 'https://bitbucket.org/acme/billing/pull-requests/87',
      draft: false,
    },
  ],
  searched: '2026-10-21T00:30:00Z',
};

/** A made-up user's nudge state at the tests' time: one agent's question with options, one without, and two sessions. */
export const nudge: State = {
  questions: [
    {
      id: '5f0c7d3e-8a1b-4c52-9d6e-2b7a1c4e9f03',
      project: 'atlas (work-laptop)',
      text: 'Merge the pull request?',
      details: 'CI is green and two reviewers approved it.',
      options: ['Yes', 'No'],
      asked_at: '2026-10-21T00:38:00Z',
    },
    {
      id: '9a2e4b60-3c7d-4f18-8e5a-7d0b6c1f2a84',
      project: 'notes (home)',
      text: 'What should the weekly review be called?',
      details: '',
      options: [],
      asked_at: '2026-10-21T00:30:00Z',
    },
  ],
  sessions: [
    {
      id: 's1',
      project: 'atlas (work-laptop)',
      note: 'Waiting on your merge decision',
      noted_at: '2026-10-21T00:38:00Z',
    },
    { id: 's2', project: 'notes (home)', note: 'Drafting the review template', noted_at: '2026-10-21T00:36:00Z' },
  ],
};

/** A made-up user's time report at the tests' time: the last 7 days, with nothing logged on the Friday or the weekend. */
export const report: Report = {
  days: [
    { date: '2026-10-15', loggedHours: 7.5, missing: false },
    { date: '2026-10-16', loggedHours: 0, missing: true },
    { date: '2026-10-17', loggedHours: 0, missing: false },
    { date: '2026-10-18', loggedHours: 0, missing: false },
    { date: '2026-10-19', loggedHours: 8, missing: false },
    { date: '2026-10-20', loggedHours: 6.25, missing: false },
    { date: '2026-10-21', loggedHours: 1.5, missing: false },
  ],
  week: { loggedHours: 15.75, targetHours: 40 },
  searched: '2026-10-21T00:38:00Z',
};
