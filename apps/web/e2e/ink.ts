import type { Page } from '@playwright/test';

// The colour --ink-muted gives text in the page's theme.
export function inkMuted(page: Page) {
  return page.evaluate(() => {
    const probe = document.body.appendChild(document.createElement('span'));
    probe.style.color = 'var(--ink-muted)';
    const { color } = getComputedStyle(probe);
    probe.remove();
    return color;
  });
}
