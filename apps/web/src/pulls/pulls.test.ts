import { afterEach, expect, test, vi } from 'vitest';

import { loadPulls, loadReviews, type Reviewer, reviewLine, type Snapshot } from './pulls';

afterEach(() => {
  vi.unstubAllGlobals();
});

test.each([
  { name: 'my open pull requests', load: loadPulls, url: '/api/pulls/mine' },
  { name: 'the open pull requests that wait for my review', load: loadReviews, url: '/api/pulls/reviews' },
])('loads $name from the pulls service', async ({ load, url }) => {
  const snapshot: Snapshot = {
    pulls: [{ id: 42, title: 'Show the order history', repo: 'acme/web', draft: false }],
    searched: '2026-10-26T00:41:00Z',
  };
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(Response.json(snapshot));
  vi.stubGlobal('fetch', fetchMock);

  expect(await load()).toEqual({ kind: 'loaded', snapshot });
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith(url);
});

test.each<{ reviewers?: Reviewer[]; want: string }>([
  { reviewers: undefined, want: 'No reviewers' },
  { reviewers: [], want: 'No reviewers' },
  { reviewers: [{ name: 'Ana', verdict: 'approved' }], want: 'Approved by Ana' },
  { reviewers: [{ name: 'Ben', verdict: 'pending' }], want: 'Waiting for Ben' },
  { reviewers: [{ name: 'Cy', verdict: 'changes_requested' }], want: 'Changes requested by Cy' },
  {
    reviewers: [
      { name: 'Ana', verdict: 'approved' },
      { name: 'Dee', verdict: 'pending' },
      { name: 'Cy', verdict: 'changes_requested' },
      { name: 'Ben', verdict: 'pending' },
    ],
    want: 'Changes requested by Cy · Waiting for Dee and Ben · Approved by Ana',
  },
  {
    reviewers: [
      { name: 'Dee', verdict: 'approved' },
      { name: 'Ben', verdict: 'changes_requested' },
      { name: 'Ana', verdict: 'approved' },
      { name: 'Cy', verdict: 'changes_requested' },
    ],
    want: 'Changes requested by Ben and Cy · Approved by Dee and Ana',
  },
  {
    reviewers: [
      { name: 'Cy', verdict: 'pending' },
      { name: 'Ana', verdict: 'pending' },
      { name: 'Ben', verdict: 'pending' },
    ],
    want: 'Waiting for Cy, Ana and Ben',
  },
])('says "$want" for where a PR stands with its reviewers', ({ reviewers, want }) => {
  expect(reviewLine(reviewers)).toBe(want);
});
