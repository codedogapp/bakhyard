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

function show(polled: Polled<Snapshot>) {
  return render(PolledSection<Snapshot, string>, {
    props: {
      polled,
      list: (snapshot) => snapshot.items,
      title: 'Assigned',
      accent: 'issues',
      noun: 'issue',
      empty: 'No open issues.',
      failed: "Couldn't load the issues.",
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
