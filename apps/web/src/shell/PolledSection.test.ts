import { createRawSnippet } from 'svelte';
import { render, screen, within } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import PolledSection from './PolledSection.svelte';
import type { Polled, Polls } from './snapshot';

// A list that keeps its items under a key of its own, as GitHub's lists do.
interface Snapshot extends Polls {
  items?: string[];
}

const searched = '2026-10-26T00:41:00Z';
const row = createRawSnippet<[string]>((item) => ({ render: () => `<span>${item()}</span>` }));

function show(polled: Polled<Snapshot>, markStale?: boolean) {
  return render(PolledSection<Snapshot, string>, {
    props: {
      polled,
      list: (snapshot) => snapshot.items,
      title: 'Assigned',
      accent: 'issues',
      noun: 'issue',
      empty: 'No open issues.',
      failed: "Couldn't load the issues.",
      markStale,
      row,
    },
  });
}

test('draws each item of the list with its row, under a head that counts them with the noun', () => {
  show({ kind: 'loaded', snapshot: { items: ['First', 'Second'], searched } });

  expect(screen.getByRole('heading', { level: 2, name: 'Assigned issues' })).toBeInTheDocument();
  expect(screen.getByText('2 issues')).toBeInTheDocument();
  expect(
    within(screen.getByRole('list'))
      .getAllByRole('listitem')
      .map((item) => item.textContent)
  ).toEqual(['First', 'Second']);
});

test('says when the items are from with the noun', () => {
  show({ kind: 'loaded', snapshot: { items: ['First'], searched, failure: 'search: 502' } });

  expect(screen.getByText(/^The last search failed/)).toHaveTextContent(
    'The last search failed: search: 502. These issues are from 26 October at 09:41.'
  );
});

test.each<Snapshot>([
  { searched, failure: 'search: 502' },
  { items: ['First'], failure: 'search: 502' },
])("doesn't say when the items are from without both a list and its time: %o", (snapshot) => {
  show({ kind: 'loaded', snapshot });

  expect(screen.getByText(/^The last search failed/)).toHaveTextContent(/^The last search failed: search: 502\.$/);
  expect(document.querySelector('time')).not.toBeInTheDocument();
});

test('marks a list from an earlier search with a Stale chip after its count, when asked to', () => {
  show({ kind: 'loaded', snapshot: { items: ['First'], searched, failure: 'search: 502' } }, true);

  expect(screen.getByText('1 issue')).toContainElement(screen.getByText('Stale'));
});

test.each<{ snapshot: Snapshot; markStale?: boolean }>([
  { snapshot: { items: ['First'], searched, failure: 'search: 502' } },
  { snapshot: { items: ['First'], searched }, markStale: true },
  { snapshot: { searched, failure: 'search: 502' }, markStale: true },
  { snapshot: { items: ['First'], failure: 'search: 502' }, markStale: true },
])('marks no list stale but one from an earlier search, when asked to: %o', ({ snapshot, markStale }) => {
  show({ kind: 'loaded', snapshot }, markStale);

  expect(screen.queryByText('Stale')).not.toBeInTheDocument();
});
