import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import Sessions from './Sessions.svelte';

test('lists each session with its note and when it was set', () => {
  render(Sessions, {
    sessions: [
      { id: 's1', project: 'atlas (work-laptop)', note: 'Tests are green', noted_at: '2026-10-21T00:41:00Z' },
      { id: 's2', project: 'notes (home)', note: '', noted_at: '0001-01-01T00:00:00Z' },
    ],
  });

  const rows = screen.getAllByRole('listitem').map((row) => row.textContent?.replace(/\s+/g, ' ').trim());
  expect(rows).toEqual(['atlas (work-laptop) Tests are green 21 October at 09:41', 'notes (home) No report yet']);
});

test('says when no agent has reported', () => {
  render(Sessions, { sessions: [] });

  expect(screen.getByText('No agent has reported.')).toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
});
