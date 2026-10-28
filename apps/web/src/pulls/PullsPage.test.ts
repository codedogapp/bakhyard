import { render, screen } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import PullsPage from './PullsPage.svelte';
import type { Snapshot } from './pulls';

afterEach(() => {
  vi.unstubAllGlobals();
});

function answer(response: Promise<Response>) {
  const fetchMock = vi.fn<typeof fetch>().mockReturnValue(response);
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const snapshot: Snapshot = {
  pulls: [
    {
      id: 42,
      title: 'Show the order history',
      repo: 'acme/web',
      url: 'https://bitbucket.org/acme/web/pull-requests/42',
      draft: false,
    },
  ],
  searched: '2026-10-26T00:41:00Z',
};

test('loads my open PRs as it shows, under its head', async () => {
  const fetchMock = answer(Promise.resolve(Response.json(snapshot)));
  render(PullsPage);

  expect(screen.getByRole('heading', { level: 1, name: 'Pull requests' })).toBeInTheDocument();
  expect(await screen.findByRole('link', { name: /^#42 / })).toHaveAttribute('href', snapshot.pulls?.[0].url);
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/pulls/mine');
});

test('shows nothing under its head while my PRs load', () => {
  answer(Promise.withResolvers<Response>().promise);
  render(PullsPage);

  expect(screen.getByRole('heading', { level: 1, name: 'Pull requests' })).toBeInTheDocument();
  expect(screen.queryByRole('heading', { level: 2 })).not.toBeInTheDocument();
  expect(screen.queryByText(/Bitbucket|load/)).not.toBeInTheDocument();
});

test("says that Bitbucket isn't set up when the service answers 404, with no section", async () => {
  answer(Promise.resolve(new Response('Bitbucket is not set up\n', { status: 404 })));
  render(PullsPage);

  expect(await screen.findByText("Bitbucket isn't set up.")).toBeInTheDocument();
  expect(screen.queryByRole('heading', { level: 2 })).not.toBeInTheDocument();
});

test('says when my PRs could not load, in their section', async () => {
  answer(Promise.resolve(new Response('Bad Gateway\n', { status: 502 })));
  render(PullsPage);

  expect(await screen.findByText("Couldn't load the pull requests.")).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 2, name: 'Your PRs' })).toBeInTheDocument();
  expect(screen.queryByText("Bitbucket isn't set up.")).not.toBeInTheDocument();
});
