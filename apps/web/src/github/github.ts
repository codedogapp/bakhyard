import { loadSnapshot, type Polled, type Polls } from '../shell/snapshot';

/** An open pull request or issue on GitHub, as the github service lists it (decision 204). */
export interface Item {
  /** The PR's or the issue's number, unique only within its repository. */
  id: number;
  title: string;
  /** The full name of the repository, such as acme/web. */
  repo: string;
  /** Its page on GitHub; none when GitHub's link to it isn't https. */
  url?: string;
  /** Whether it's a draft PR; an issue never is. */
  draft: boolean;
}

/**
 * The body of GET /api/github/mine, /api/github/reviews and /api/github/issues: what the github service last found
 * on GitHub (decision 204).
 */
export interface Snapshot extends Polls {
  /** The open PRs or issues from the last search that succeeded, the latest updated first; none before the first. */
  items?: Item[];
}

/** A list of the GitHub page: loading, off when GitHub isn't set up, failed to load, or loaded. */
export type Items = Polled<Snapshot>;

/** The GitHub page's lists: my open PRs, the open PRs that wait for my review, and the open issues assigned to me. */
export type List = 'mine' | 'reviews' | 'issues';

/** Loads one of the lists; never fails, since a failed load is one of the outcomes. */
export function loadItems(list: List): Promise<Items> {
  return loadSnapshot(`/api/github/${list}`);
}
