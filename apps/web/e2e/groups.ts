import type { Page } from '@playwright/test';

/** The heads of the Todos page's priority groups, in order. */
export function heads(page: Page) {
  return page.getByRole('main').getByRole('heading', { level: 2 });
}

/** The section of the Todos page's priority group with the head. */
export function group(page: Page, head: string) {
  return page
    .getByRole('main')
    .locator('section')
    .filter({ has: page.getByRole('heading', { level: 2, name: head }) });
}
