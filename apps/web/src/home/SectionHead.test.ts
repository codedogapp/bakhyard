import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import SectionHead from './SectionHead.svelte';

test('shows the title with its accent word, and the note', () => {
  render(SectionHead, { props: { title: 'Needs you', accent: 'today', note: '6 items · most urgent first' } });

  expect(screen.getByRole('heading', { level: 2, name: 'Needs you today' })).toBeInTheDocument();
  expect(screen.getByText('today').tagName).toBe('EM');
  expect(screen.getByRole('paragraph')).toHaveTextContent('6 items · most urgent first');
});

test('shows no note without one', () => {
  render(SectionHead, { props: { title: 'Your', accent: 'apps' } });

  expect(screen.getByRole('heading', { level: 2, name: 'Your apps' })).toBeInTheDocument();
  expect(screen.queryByRole('paragraph')).not.toBeInTheDocument();
});
