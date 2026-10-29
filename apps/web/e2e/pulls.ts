import type { Page } from '@playwright/test';

import type { Snapshot } from '../src/pulls/pulls';
import { group } from './groups';

/** Answers the PRs page's call to the URL with the snapshot, or with the status. */
async function answer(page: Page, url: string, snapshot: Snapshot | number) {
  await page.route(url, (route) =>
    typeof snapshot === 'number' ? route.fulfill({ status: snapshot }) : route.fulfill({ json: snapshot })
  );
}

/** Answers the PRs page's call for my PRs with the snapshot, or with the status. */
export async function answerPulls(page: Page, snapshot: Snapshot | number) {
  await answer(page, '/api/pulls/mine', snapshot);
}

/** Answers the PRs page's call for the PRs that wait for my review with the snapshot, or with the status. */
export async function answerReviews(page: Page, snapshot: Snapshot | number) {
  await answer(page, '/api/pulls/reviews', snapshot);
}

/** The PRs page's section of my PRs. */
export function mine(page: Page) {
  return group(page, 'Your PRs');
}

/** The PRs page's section of the PRs that wait for my review. */
export function reviews(page: Page) {
  return group(page, 'Review requests');
}
