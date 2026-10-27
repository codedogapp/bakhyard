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

/**
 * Notes the Todos page's heads the moment its first todo shows, in the `data-heads` attribute of its `html` element, so a
 * test can check that nothing the todos wait for pushed them down after they showed.
 */
export async function noteFirstHeads(page: Page) {
  await page.addInitScript(() => {
    new MutationObserver((_, observer) => {
      if (document.querySelector('main input[type="checkbox"]')) {
        const shown = [...document.querySelectorAll('main section h2')].map((head) => head.textContent);
        document.documentElement.dataset.heads = JSON.stringify(shown);
        observer.disconnect();
      }
    }).observe(document, { childList: true, subtree: true });
  });
}
