import { render, screen } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import PullsPage from './PullsPage.svelte';
import type { Snapshot } from './pulls';

afterEach(() => {
  vi.unstubAllGlobals();
});

// The page's answers: my PRs', then the review requests'.
type Answers = [mine: () => Promise<Response>, reviews: () => Promise<Response>];

function answer([mine, reviews]: Answers) {
  const fetchMock = vi.fn<typeof fetch>(async (url) => (url === '/api/pulls/mine' ? mine() : reviews()));
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const searched = '2026-10-26T00:41:00Z';
const history: Snapshot = {
  pulls: [
    {
      id: 42,
      title: 'Show the order history',
      repo: 'acme/web',
      url: 'https://bitbucket.org/acme/web/pull-requests/42',
      draft: false,
    },
  ],
  searched,
};
const certificates: Snapshot = {
  pulls: [
    {
      id: 7,
      title: 'Rotate the certificates',
      repo: 'acme/ops',
      url: 'https://bitbucket.org/acme/ops/pull-requests/7',
      draft: false,
    },
  ],
  searched,
};

const loaded = (snapshot: Snapshot) => async () => Response.json(snapshot);
const loading = () => Promise.withResolvers<Response>().promise;
const off = async () => new Response('Bitbucket is not set up\n', { status: 404 });
const failed = async () => new Response('Bad Gateway\n', { status: 502 });

const heads = () => screen.queryAllByRole('heading', { level: 2 }).map((head) => head.textContent);

test('loads my open PRs and the PRs that wait for my review as it shows, mine first', async () => {
  const fetchMock = answer([loaded(history), loaded(certificates)]);
  render(PullsPage);

  expect(screen.getByRole('heading', { level: 1, name: 'Pull requests' })).toBeInTheDocument();
  expect(await screen.findByRole('link', { name: /^#42 / })).toHaveAttribute('href', history.pulls?.[0].url);
  expect(await screen.findByRole('link', { name: /^#7 / })).toHaveAttribute('href', certificates.pulls?.[0].url);
  expect(heads()).toEqual(['Your PRs', 'Review requests']);
  expect(fetchMock.mock.calls).toEqual([['/api/pulls/mine'], ['/api/pulls/reviews']]);
});

test('says where each of my PRs stands with its reviewers, but not the review requests', async () => {
  answer([loaded(history), loaded(certificates)]);
  render(PullsPage);

  expect(await screen.findByRole('link', { name: /^#42 .* acme\/web No reviewers$/ })).toBeInTheDocument();
  expect(await screen.findByRole('link', { name: /^#7 .* acme\/ops$/ })).toBeInTheDocument();
  expect(screen.getAllByText('No reviewers')).toHaveLength(1);
});

test('shows nothing under its head while the lists load', () => {
  answer([loading, loading]);
  render(PullsPage);

  expect(screen.getByRole('heading', { level: 1, name: 'Pull requests' })).toBeInTheDocument();
  expect(heads()).toEqual([]);
  expect(screen.queryByText(/Bitbucket|load/)).not.toBeInTheDocument();
});

test.each<{ name: string; answers: Answers; link: RegExp; head: string }>([
  { name: 'my PRs', answers: [loaded(history), loading], link: /^#42 /, head: 'Your PRs' },
  { name: 'the review requests', answers: [loading, loaded(certificates)], link: /^#7 /, head: 'Review requests' },
])('shows $name once they load, without waiting for the other list', async ({ answers, link, head }) => {
  answer(answers);
  render(PullsPage);

  expect(await screen.findByRole('link', { name: link })).toBeInTheDocument();
  expect(heads()).toEqual([head]);
});

test("says that Bitbucket isn't set up when both lists answer 404, with no section", async () => {
  answer([off, off]);
  render(PullsPage);

  expect(await screen.findByText("Bitbucket isn't set up.")).toBeInTheDocument();
  expect(heads()).toEqual([]);
});

// A pulls service older than the review requests answers 404 for those alone (decision 172).
test.each<{ name: string; answers: Answers; link: RegExp; head: string }>([
  { name: 'my PRs', answers: [loaded(history), off], link: /^#42 /, head: 'Your PRs' },
  { name: 'the review requests', answers: [off, loaded(certificates)], link: /^#7 /, head: 'Review requests' },
])('shows $name alone when only the other list answers 404', async ({ answers, link, head }) => {
  answer(answers);
  render(PullsPage);

  expect(await screen.findByRole('link', { name: link })).toBeInTheDocument();
  expect(heads()).toEqual([head]);
  expect(screen.queryByText("Bitbucket isn't set up.")).not.toBeInTheDocument();
});

test.each<{ name: string; answers: Answers; says: string; link: RegExp }>([
  { name: 'my PRs', answers: [failed, loaded(certificates)], says: "Couldn't load the pull requests.", link: /^#7 / },
  {
    name: 'the review requests',
    answers: [loaded(history), failed],
    says: "Couldn't load the review requests.",
    link: /^#42 /,
  },
])('says when $name could not load, in their section, beside the other list', async ({ answers, says, link }) => {
  answer(answers);
  render(PullsPage);

  expect(await screen.findByText(says)).toBeInTheDocument();
  expect(await screen.findByRole('link', { name: link })).toBeInTheDocument();
  expect(heads()).toEqual(['Your PRs', 'Review requests']);
  expect(screen.queryByText("Bitbucket isn't set up.")).not.toBeInTheDocument();
});
