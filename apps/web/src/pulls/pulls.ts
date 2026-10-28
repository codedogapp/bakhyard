import { loadSnapshot, type Polled, type Polls } from '../shell/snapshot';

/** One of my open pull requests on Bitbucket, as the pulls service lists it. */
export interface Pull {
  /** The PR's number, unique only within its repository. */
  id: number;
  title: string;
  /** The full name of the repository that the PR merges into, such as acme/web. */
  repo: string;
  /** The PR's page on Bitbucket; none when Bitbucket's link to it isn't https. */
  url?: string;
  draft: boolean;
}

/** The body of GET /api/pulls/mine: what the pulls service last found on Bitbucket (decision 146). */
export interface Snapshot extends Polls {
  /** My open PRs, from the last poll that succeeded, the latest updated first; none before the first. */
  pulls?: Pull[];
}

/** The PRs page's PRs: loading, off when Bitbucket isn't set up, failed to load, or loaded. */
export type Pulls = Polled<Snapshot>;

/** Loads my open PRs; never fails, since a failed load is one of the outcomes. */
export function loadPulls(): Promise<Pulls> {
  return loadSnapshot('/api/pulls/mine');
}

/** The PRs' count: "1 PR" or "3 PRs"; none without PRs, which the list itself says. */
export function pullsNote(count: number): string | undefined {
  if (count === 0) {
    return undefined;
  }
  return count === 1 ? '1 PR' : `${count} PRs`;
}
