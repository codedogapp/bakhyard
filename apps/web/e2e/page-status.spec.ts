import type { Page } from '@playwright/test';

import { home, tickets } from './demo';
import { expect, test } from './fixtures';
import { answerGitHub } from './github';
import { inkMuted } from './ink';
import { answerDismissals, answerTickets } from './jira';
import { answerPulls, answerReviews } from './pulls';

// Each page's line in place of its sections, and the answers that make the page show it.
const lines: { name: string; path: string; says: string; answer: (page: Page) => Promise<void> }[] = [
  {
    name: 'PRs',
    path: '/pulls',
    says: "Bitbucket isn't set up.",
    answer: async (page) => {
      await answerPulls(page, 404);
      await answerReviews(page, 404);
    },
  },
  {
    name: 'Time',
    path: '/time',
    says: "Jira isn't set up.",
    answer: async (page) => {
      await answerTickets(page, tickets);
      await page.route('/api/time/report', (route) => route.fulfill({ status: 404 }));
    },
  },
  {
    name: 'GitHub',
    path: '/github',
    says: "GitHub isn't set up.",
    answer: async (page) => {
      for (const list of ['mine', 'reviews', 'issues'] as const) {
        await answerGitHub(page, list, 404);
      }
    },
  },
  {
    name: 'Games',
    path: '/games',
    says: "Exophase isn't set up.",
    answer: async (page) => {
      await page.route('/api/games/recent', (route) => route.fulfill({ status: 404 }));
    },
  },
  {
    name: 'Planner',
    path: '/planner',
    says: "Copilot isn't set up.",
    answer: async (page) => {
      await page.route('/api/planner/plans', (route) => route.fulfill({ status: 404 }));
    },
  },
  {
    name: 'Nudge',
    path: '/nudge',
    says: "Couldn't load the questions.",
    answer: async (page) => {
      await page.route('/api/nudge/state', (route) => route.fulfill({ status: 502 }));
      await page.route('/api/nudge/events', (route) =>
        route.fulfill({ contentType: 'text/event-stream', body: 'retry: 600000\n: open\n\n' })
      );
    },
  },
  {
    name: 'Todos',
    path: '/todos',
    says: "Couldn't load the todos.",
    answer: async (page) => {
      await page.route('/api/todos', (route) => route.fulfill({ status: 500, body: 'Internal Server Error\n' }));
      await answerDismissals(page);
      await answerTickets(page, tickets);
    },
  },
  {
    name: 'Alerts',
    path: '/alerts',
    says: "Couldn't load the alerts.",
    answer: async (page) => {
      await page.route('/api/notify/alerts', (route) =>
        route.fulfill({ status: 500, body: 'Internal Server Error\n' })
      );
    },
  },
];

for (const { name, path, says, answer } of lines) {
  test(`the ${name} page's status line is muted, with a section's space below it`, async ({ page }, testInfo) => {
    await page.route('/api/home', (route) => route.fulfill({ json: home }));
    await answer(page);
    await page.goto(path);

    const line = page.getByText(says);
    await expect(line).toHaveCSS('color', await inkMuted(page));
    await expect(line).toHaveCSS('padding-bottom', testInfo.project.name === 'phone' ? '48px' : '64px');
  });
}
