import { render, screen, within } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import GitHubPage from './GitHubPage.svelte';
import type { List, Snapshot } from './github';

afterEach(() => {
  vi.unstubAllGlobals();
});

type Answer = () => Promise<Response>;

// Answers each list's call with its answer, or leaves it loading.
function answer(answers: Partial<Record<List, Answer>>) {
  const fetchMock = vi.fn<typeof fetch>(async (url) => {
    const list = String(url).replace('/api/github/', '') as List;
    return (answers[list] ?? loading)();
  });
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const searched = '2026-10-26T00:41:00Z';
const mine: Snapshot = {
  items: [
    {
      id: 42,
      title: 'Show the order history',
      repo: 'acme/web',
      url: 'https://github.com/acme/web/pull/42',
      draft: false,
    },
    {
      id: 9,
      title: 'Try the new cache',
      repo: 'me/dotfiles',
      url: 'https://github.com/me/dotfiles/pull/9',
      draft: true,
    },
  ],
  searched,
};
const reviews: Snapshot = {
  items: [
    {
      id: 7,
      title: 'Rotate the certificates',
      repo: 'acme/ops',
      url: 'https://github.com/acme/ops/pull/7',
      draft: false,
    },
  ],
  searched,
};
const issues: Snapshot = {
  items: [{ id: 3, title: 'The build fails on Windows', repo: 'acme/cli', draft: false }],
  searched,
};

const loaded = (snapshot: Snapshot) => async () => Response.json(snapshot);
const loading = () => Promise.withResolvers<Response>().promise;
const off = async () => new Response('GitHub is not set up\n', { status: 404 });
const failed = async () => new Response('Bad Gateway\n', { status: 502 });

const heads = () => screen.queryAllByRole('heading', { level: 2 }).map((head) => head.textContent);
const section = (head: string) =>
  screen.getByRole('heading', { level: 2, name: head }).closest('section') as HTMLElement;

test('loads my PRs, the review requests and my issues as it shows, in that order', async () => {
  const fetchMock = answer({ mine: loaded(mine), reviews: loaded(reviews), issues: loaded(issues) });
  render(GitHubPage);

  expect(screen.getByRole('heading', { level: 1, name: 'GitHub' })).toBeInTheDocument();
  expect(await screen.findByRole('link', { name: /^#42 Show the order history acme\/web$/ })).toHaveAttribute(
    'href',
    'https://github.com/acme/web/pull/42'
  );
  expect(await screen.findByRole('link', { name: /^#7 Rotate the certificates acme\/ops$/ })).toBeInTheDocument();
  expect(await screen.findByText('The build fails on Windows')).toBeInTheDocument();
  expect(heads()).toEqual(['Your PRs', 'Review requests', 'Assigned issues']);
  expect(fetchMock.mock.calls).toEqual([['/api/github/mine'], ['/api/github/reviews'], ['/api/github/issues']]);
});

test('counts each list, marks a draft, and shows an item without a link in a row that is not one', async () => {
  answer({ mine: loaded(mine), reviews: loaded(reviews), issues: loaded(issues) });
  render(GitHubPage);

  expect(await screen.findByRole('link', { name: /^#9 Try the new cache me\/dotfiles Draft$/ })).toBeInTheDocument();
  expect(within(section('Your PRs')).getByText('2 PRs')).toBeInTheDocument();
  expect(within(section('Review requests')).getByText('1 PR')).toBeInTheDocument();
  expect(within(section('Assigned issues')).getByText('1 issue')).toBeInTheDocument();
  expect(screen.getAllByText('Draft')).toHaveLength(1);
  expect(screen.getByText('The build fails on Windows').closest('a')).toBeNull();
  expect(screen.getAllByRole('link')).toHaveLength(3);
});

test('shows nothing under its head while the lists load', () => {
  answer({});
  render(GitHubPage);

  expect(screen.getByRole('heading', { level: 1, name: 'GitHub' })).toBeInTheDocument();
  expect(heads()).toEqual([]);
  expect(screen.queryByText(/GitHub isn't|load/)).not.toBeInTheDocument();
});

test.each<{ list: List; snapshot: Snapshot; head: string }>([
  { list: 'mine', snapshot: mine, head: 'Your PRs' },
  { list: 'reviews', snapshot: reviews, head: 'Review requests' },
  { list: 'issues', snapshot: issues, head: 'Assigned issues' },
])('shows the $list list once it loads, without waiting for the others', async ({ list, snapshot, head }) => {
  answer({ [list]: loaded(snapshot) });
  render(GitHubPage);

  expect(await screen.findByText(snapshot.items?.[0].title ?? '')).toBeInTheDocument();
  expect(heads()).toEqual([head]);
});

test("says that GitHub isn't set up when every list answers 404, with no section", async () => {
  answer({ mine: off, reviews: off, issues: off });
  render(GitHubPage);

  expect(await screen.findByText("GitHub isn't set up.")).toHaveClass('page-status');
  expect(heads()).toEqual([]);
});

test('leaves out only the section of a list that answers 404 when the others load', async () => {
  answer({ mine: off, reviews: loaded(reviews), issues: loaded(issues) });
  render(GitHubPage);

  expect(await screen.findByText('The build fails on Windows')).toBeInTheDocument();
  await vi.waitFor(() => expect(heads()).toEqual(['Review requests', 'Assigned issues']));
  expect(screen.queryByText("GitHub isn't set up.")).not.toBeInTheDocument();
});

test.each<{ list: List; head: string; emptyText: string }>([
  { list: 'mine', head: 'Your PRs', emptyText: 'No open pull requests.' },
  { list: 'reviews', head: 'Review requests', emptyText: 'No pull requests to review.' },
  { list: 'issues', head: 'Assigned issues', emptyText: 'No open issues.' },
])('says when the $list list is empty, in its section', async ({ list, head, emptyText }) => {
  answer({
    mine: loaded(mine),
    reviews: loaded(reviews),
    issues: loaded(issues),
    [list]: loaded({ items: [], searched }),
  });
  render(GitHubPage);

  expect(await screen.findByText(emptyText)).toBeInTheDocument();
  expect(within(section(head)).getByText(emptyText)).toBeInTheDocument();
  expect(screen.getAllByText(/^No /)).toHaveLength(1);
});

test.each<{ list: List; head: string; failedText: string }>([
  { list: 'mine', head: 'Your PRs', failedText: "Couldn't load the pull requests." },
  { list: 'reviews', head: 'Review requests', failedText: "Couldn't load the review requests." },
  { list: 'issues', head: 'Assigned issues', failedText: "Couldn't load the issues." },
])('says when the $list list could not load, in its section, beside the others', async ({ list, head, failedText }) => {
  answer({ mine: loaded(mine), reviews: loaded(reviews), issues: loaded(issues), [list]: failed });
  render(GitHubPage);

  expect(await screen.findByText(failedText)).toBeInTheDocument();
  expect(within(section(head)).getByText(failedText)).toBeInTheDocument();
  await vi.waitFor(() => expect(heads()).toEqual(['Your PRs', 'Review requests', 'Assigned issues']));
  expect(screen.getAllByText(/^Couldn't load/)).toHaveLength(1);
  expect(screen.queryByText("GitHub isn't set up.")).not.toBeInTheDocument();
});
