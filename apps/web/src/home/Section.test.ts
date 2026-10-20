import { createRawSnippet } from 'svelte';
import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import Section from './Section.svelte';

const children = createRawSnippet(() => ({ render: () => '<div>The content</div>' }));

test('shows the title with its accent word, the note and the content', () => {
  render(Section, { props: { title: 'Needs you', accent: 'today', note: '6 items · most urgent first', children } });

  expect(screen.getByRole('heading', { level: 2, name: 'Needs you today' })).toBeInTheDocument();
  expect(screen.getByText('today').tagName).toBe('EM');
  expect(screen.getByRole('paragraph')).toHaveTextContent('6 items · most urgent first');
  expect(screen.getByText('The content')).toBeInTheDocument();
});

test('shows no note without one', () => {
  render(Section, { props: { title: 'Your', accent: 'apps', children } });

  expect(screen.getByRole('heading', { level: 2, name: 'Your apps' })).toBeInTheDocument();
  expect(screen.queryByRole('paragraph')).not.toBeInTheDocument();
});
