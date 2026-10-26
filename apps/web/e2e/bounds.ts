import type { Locator } from '@playwright/test';

/** The box of what the locator finds, which shows. */
export async function bounds(locator: Locator) {
  const found = await locator.boundingBox();
  if (!found) {
    throw new Error(`${locator} has no box`);
  }
  return found;
}
