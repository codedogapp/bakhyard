import type { Home } from '../src/home/api';

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
