import { loadSnapshot, type Polled, type Polls } from '../shell/snapshot';

/** Where someone who reviews a PR stands (decision 173). */
type Verdict = 'pending' | 'approved' | 'changes_requested';

/** Someone asked to review a PR of mine, or anyone else who has approved it or requested changes. */
export interface Reviewer {
  name: string;
  verdict: Verdict;
}

/** An open pull request on Bitbucket, as the pulls service lists it. */
export interface Pull {
  /** The PR's number, unique only within its repository. */
  id: number;
  title: string;
  /** The full name of the repository that the PR merges into, such as acme/web. */
  repo: string;
  /** The PR's page on Bitbucket; none when Bitbucket's link to it isn't https. */
  url?: string;
  draft: boolean;
  /** Who reviews the PR, in Bitbucket's order; only for my PRs, and none when nobody does. */
  reviewers?: Reviewer[];
}

/**
 * The body of GET /api/pulls/mine and of GET /api/pulls/reviews: what the pulls service last found on Bitbucket
 * (decisions 146 and 171).
 */
export interface Snapshot extends Polls {
  /** The open PRs from the last poll that succeeded, the latest updated first; none before the first. */
  pulls?: Pull[];
}

/** A list of the PRs page: loading, off when Bitbucket isn't set up, failed to load, or loaded. */
export type Pulls = Polled<Snapshot>;

/** Loads my open PRs; never fails, since a failed load is one of the outcomes. */
export function loadPulls(): Promise<Pulls> {
  return loadSnapshot('/api/pulls/mine');
}

/** Loads the open PRs that wait for my review; never fails, as loadPulls. */
export function loadReviews(): Promise<Pulls> {
  return loadSnapshot('/api/pulls/reviews');
}

const names = new Intl.ListFormat('en-GB', { type: 'conjunction' });

// The review line's groups, the most urgent first: what I must answer, then whom to remind, then who is done.
const groups: readonly { verdict: Verdict; words: string }[] = [
  { verdict: 'changes_requested', words: 'Changes requested by' },
  { verdict: 'pending', words: 'Waiting for' },
  { verdict: 'approved', words: 'Approved by' },
];

/**
 * Where my PR stands with its reviewers, such as "Changes requested by Cy · Waiting for Ben and Dee · Approved by
 * Ana": a group for each verdict that someone has given, naming them in the reviewers' order (decision 174).
 */
export function reviewLine(reviewers: readonly Reviewer[] = []): string {
  const parts = groups.flatMap(({ verdict, words }) => {
    const named = reviewers.filter((reviewer) => reviewer.verdict === verdict).map(({ name }) => name);
    return named.length > 0 ? [`${words} ${names.format(named)}`] : [];
  });
  return parts.length > 0 ? parts.join(' · ') : 'No reviewers';
}
