import type { Page } from '@playwright/test';

/** The colour that the token, such as --fill, gives text in the page's theme. */
export function colour(page: Page, token: string) {
  return page.evaluate((name) => {
    const probe = document.body.appendChild(document.createElement('span'));
    probe.style.color = `var(${name})`;
    const { color } = getComputedStyle(probe);
    probe.remove();
    return color;
  }, token);
}

// The colour --ink-muted gives text in the page's theme.
export function inkMuted(page: Page) {
  return colour(page, '--ink-muted');
}
