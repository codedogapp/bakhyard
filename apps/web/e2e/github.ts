import type { Page } from '@playwright/test';

import type { List, Snapshot } from '../src/github/github';

/**
 * A made-up user's open PRs on GitHub, the latest updated first, as the github service found them two minutes before
 * the tests' time.
 */
export const githubMine: Snapshot = {
  items: [
    {
      id: 214,
      title: 'Add a dark theme to the docs site',
      repo: 'acme/docs',
      url: 'https://github.com/acme/docs/pull/214',
      draft: false,
    },
    {
      id: 12,
      title: 'Share the shell config across machines',
      repo: 'mira/dotfiles',
      url: 'https://github.com/mira/dotfiles/pull/12',
      draft: true,
    },
  ],
  searched: '2026-10-21T00:39:00Z',
};

/** The open PRs on GitHub that wait for a made-up user's review, as the github service found them with githubMine. */
export const githubReviews: Snapshot = {
  items: [
    {
      id: 14,
      title: 'Speed up the prompt in large repos',
      repo: 'kai/dotfiles',
      url: 'https://github.com/kai/dotfiles/pull/14',
      draft: false,
    },
  ],
  searched: '2026-10-21T00:39:00Z',
};

/** The open issues on GitHub assigned to a made-up user, as the github service found them with githubMine. */
export const githubIssues: Snapshot = {
  items: [
    {
      id: 88,
      title: 'The CLI crashes on an empty config',
      repo: 'acme/cli',
      url: 'https://github.com/acme/cli/issues/88',
      draft: false,
    },
    {
      id: 5,
      title: 'Document the release steps',
      repo: 'acme/docs',
      url: 'https://github.com/acme/docs/issues/5',
      draft: false,
    },
  ],
  searched: '2026-10-21T00:39:00Z',
};

/** Answers the GitHub page's call for the list with the snapshot, or with the status. */
export async function answerGitHub(page: Page, list: List, snapshot: Snapshot | number) {
  await page.route(`/api/github/${list}`, (route) =>
    typeof snapshot === 'number' ? route.fulfill({ status: snapshot }) : route.fulfill({ json: snapshot })
  );
}

/** Answers each of the GitHub page's calls with the made-up user's list. */
export async function answerAllGitHub(page: Page) {
  await answerGitHub(page, 'mine', githubMine);
  await answerGitHub(page, 'reviews', githubReviews);
  await answerGitHub(page, 'issues', githubIssues);
}
