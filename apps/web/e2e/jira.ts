import type { Locator, Page } from '@playwright/test';

import type { Snapshot } from '../src/todos/tickets';

/** A todo's URL. The glob `/api/todos/*` matches the tickets' URL too, so a route for the todos must use this. */
export const aTodo = /\/api\/todos\/\d+$/;

/** Answers the Todos page's tickets call as the todos service does without Jira, with no ticket dismissed. */
export async function jiraOff(page: Page) {
  await answerDismissals(page);
  await page.route('/api/todos/tickets', (route) => route.fulfill({ status: 404, body: 'Jira is not set up\n' }));
}

/** Answers the Todos page's call for the dismissed tickets with their keys, by default none, or with the status. */
export async function answerDismissals(page: Page, dismissed: string[] | number = []) {
  await page.route('/api/todos/dismissals', (route) =>
    typeof dismissed === 'number'
      ? route.fulfill({ status: dismissed })
      : route.fulfill({ json: { dismissals: dismissed } })
  );
}

/** Answers the Todos page's tickets call with the snapshot, or with the status. */
export async function answerTickets(page: Page, snapshot: Snapshot | number) {
  await page.route('/api/todos/tickets', (route) =>
    typeof snapshot === 'number' ? route.fulfill({ status: snapshot }) : route.fulfill({ json: snapshot })
  );
}

/** How each chip looks: filled with the ink, or outlined in it with its text in the ink; else its background. */
export function chipLooks(chips: Locator): Promise<string[]> {
  return chips.evaluateAll((all) =>
    all.map((chip) => {
      const { backgroundColor, borderTopColor, color } = getComputedStyle(chip);
      if (backgroundColor === borderTopColor) {
        return 'filled';
      }
      return backgroundColor === 'rgba(0, 0, 0, 0)' && color === borderTopColor ? 'outlined' : backgroundColor;
    })
  );
}
