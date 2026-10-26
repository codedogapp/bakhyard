import { test as base, type ConsoleMessage, expect, type Page } from '@playwright/test';

export { expect };

// The time every test runs at, unless it sets its own: Wednesday 21 October 2026, 09:41 in Tokyo.
const now = new Date('2026-10-21T09:41:00+09:00');

// Browsers log an API call's error status, such as a 401, or its failure as an error. The test answered the call that
// way on purpose, since every API call is the test's to answer, and watch reports those it doesn't. A missing file's
// 404 still counts.
function aboutAnAPICall(message: ConsoleMessage): boolean {
  return URL.parse(message.location().url)?.pathname.startsWith('/api/') === true;
}

/**
 * Watches the page and returns what goes wrong in it, as it happens: the API calls that no route of the test answers,
 * which it aborts, console errors, uncaught page errors and Content Security Policy violations.
 */
export async function watch(page: Page): Promise<string[]> {
  const problems: string[] = [];
  // The route added last answers first, so this one gets only the calls that no route of the test answers.
  await page.route('/api/**', (route) => {
    problems.push(`unanswered: ${route.request().method()} ${route.request().url()}`);
    return route.abort();
  });
  page.on('console', (message) => {
    if (message.type() === 'error' && !aboutAnAPICall(message)) {
      problems.push(`console: ${message.text()}`);
    }
  });
  page.on('pageerror', (error) => problems.push(`page error: ${error.message}`));
  // Not every browser logs a violation to the console.
  await page.addInitScript(() => {
    document.addEventListener('securitypolicyviolation', (event) => {
      console.error(`CSP violation: ${event.effectiveDirective} ${event.blockedURI}`);
    });
  });
  return problems;
}

/** Playwright's test, with the clock fixed at now. Anything watch finds in the page fails the test. */
export const test = base.extend<{ problems: string[] }>({
  problems: [
    async ({ page }, use) => {
      await page.clock.setFixedTime(now);
      const problems = await watch(page);
      await use(problems);
      expect(problems, 'what went wrong in the page').toEqual([]);
    },
    { auto: true },
  ],
});
