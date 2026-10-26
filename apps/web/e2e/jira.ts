import type { Page } from '@playwright/test';

/** A todo's URL. The glob `/api/todos/*` matches the tickets' URL too, so a route for the todos must use this. */
export const aTodo = /\/api\/todos\/\d+$/;

/** Answers the Todos page's tickets call as the todos service does without Jira. */
export async function jiraOff(page: Page) {
  await page.route('/api/todos/tickets', (route) => route.fulfill({ status: 404, body: 'Jira is not set up\n' }));
}
