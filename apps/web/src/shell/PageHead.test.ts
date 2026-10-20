import { createRawSnippet } from 'svelte';
import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import PageHead from './PageHead.svelte';

const children = createRawSnippet(() => ({ render: () => '<span>Time</span>' }));

test('shows the eyebrow, the heading and the summary', () => {
  render(PageHead, { props: { eyebrow: 'Coming soon', summary: 'Hours logged on the Jira tickets', children } });

  expect(screen.getByRole('heading', { level: 1, name: 'Time' })).toBeInTheDocument();
  expect(screen.getAllByRole('paragraph').map((p) => p.textContent)).toEqual([
    'Coming soon',
    'Hours logged on the Jira tickets',
  ]);
});

test('shows only the heading without an eyebrow and a summary', () => {
  render(PageHead, { props: { children } });

  expect(screen.getByRole('heading', { level: 1, name: 'Time' })).toBeInTheDocument();
  expect(screen.queryByRole('paragraph')).not.toBeInTheDocument();
});
