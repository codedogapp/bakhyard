import { render, screen, within } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import type { Pull, Pulls } from './pulls';
import YourPulls from './YourPulls.svelte';

const history: Pull = {
  id: 42,
  title: 'Show the <b>order</b> history',
  repo: 'acme/web',
  url: 'https://bitbucket.org/acme/web/pull-requests/42',
  draft: false,
};
const certificates: Pull = {
  id: 7,
  title: 'Rotate the staging certificates',
  repo: 'acme/ops',
  url: 'https://bitbucket.org/acme/ops/pull-requests/7',
  draft: true,
};
const searched = '2026-10-26T00:41:00Z';

function show(pulls: Pulls) {
  return render(YourPulls, { props: { pulls } });
}

const text = (element: Element) => element.textContent?.replaceAll(/\s+/g, ' ').trim();

test('lists my PRs in their order under a head that counts them, each a link to the PR', () => {
  show({ kind: 'loaded', snapshot: { pulls: [history, certificates], searched } });

  expect(screen.getByRole('heading', { level: 2, name: 'Your PRs' })).toBeInTheDocument();
  expect(screen.getByText('2 PRs')).toBeInTheDocument();
  const links = within(screen.getByRole('list')).getAllByRole('link');
  expect(links.map((link) => [link.getAttribute('href'), text(link)])).toEqual([
    [history.url, '#42 Show the <b>order</b> history acme/web'],
    [certificates.url, '#7 Rotate the staging certificates acme/ops Draft'],
  ]);
  expect(screen.getByText('Draft')).toHaveClass('chip');
  expect(screen.getByText('Draft')).not.toHaveClass('chip-filled');
  expect(screen.queryByText(/failed/)).not.toBeInTheDocument();
});

test("a PR without a link to Bitbucket gets a row that isn't a link", () => {
  const { url: _, ...unlinked } = history;
  show({ kind: 'loaded', snapshot: { pulls: [unlinked, certificates], searched } });

  const rows = within(screen.getByRole('list')).getAllByRole('listitem');
  expect(rows.map(text)).toEqual([
    '#42 Show the <b>order</b> history acme/web',
    '#7 Rotate the staging certificates acme/ops Draft',
  ]);
  expect(within(rows[0]).queryByRole('link')).not.toBeInTheDocument();
  expect(rows[0].querySelector('.list-row')).not.toHaveAttribute('href');
  expect(within(rows[1]).getByRole('link')).toHaveAttribute('href', certificates.url);
});

test('says when I have no open PRs, with no count', () => {
  show({ kind: 'loaded', snapshot: { pulls: [], searched } });

  expect(screen.getByRole('heading', { level: 2, name: 'Your PRs' })).toBeInTheDocument();
  expect(screen.getByText('No open pull requests.')).toBeInTheDocument();
  expect(screen.queryByText(/^\d+ PRs?$/)).not.toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
});

test('says why the last search failed, and when the PRs it shows are from', () => {
  const failure = 'list pull requests: 502 Bad Gateway';
  show({ kind: 'loaded', snapshot: { pulls: [history], searched, failure } });

  const line = screen.getByText(/^The last search failed/);
  expect(line).toHaveTextContent(
    'The last search failed: list pull requests: 502 Bad Gateway. These PRs are from 26 October at 09:41.'
  );
  expect(within(line).getByText('26 October at 09:41')).toHaveAttribute('datetime', searched);
  expect(screen.getByText('1 PR')).toBeInTheDocument();
  expect(within(screen.getByRole('list')).getByRole('link')).toHaveAttribute('href', history.url);
});

test('says why the last search failed, with no list, before any search has succeeded', () => {
  show({ kind: 'loaded', snapshot: { failure: 'find me: 401 Unauthorized' } });

  expect(screen.getByText(/^The last search failed/)).toHaveTextContent(
    /^The last search failed: find me: 401 Unauthorized\.$/
  );
  expect(screen.queryByText('No open pull requests.')).not.toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
  expect(screen.queryByText(/^\d+ PRs?$/)).not.toBeInTheDocument();
});

test('says when the PRs could not load', () => {
  show({ kind: 'failed' });

  expect(screen.getByRole('heading', { level: 2, name: 'Your PRs' })).toBeInTheDocument();
  expect(screen.getByText("Couldn't load the pull requests.")).toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
});

test.each<Pulls>([{ kind: 'loading' }, { kind: 'off' }])('shows nothing while $kind', (pulls) => {
  const { container } = show(pulls);

  expect(container.childElementCount).toBe(0);
  expect(container.textContent).toBe('');
});
