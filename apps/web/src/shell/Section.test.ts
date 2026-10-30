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

test('shows no note and no link without them', () => {
  render(Section, { props: { title: 'Your', accent: 'apps', children } });

  expect(screen.getByRole('heading', { level: 2, name: 'Your apps' })).toBeInTheDocument();
  expect(screen.queryByRole('paragraph')).not.toBeInTheDocument();
  expect(screen.queryByRole('link')).not.toBeInTheDocument();
});

test('shows a text link, with an arrow hidden from screen readers', () => {
  render(Section, { props: { title: 'This', accent: 'week', link: { href: '/time', label: 'Day report' }, children } });

  const link = screen.getByRole('link', { name: 'Day report' });
  expect(link).toHaveAttribute('href', '/time');
  expect(link.querySelector('svg')).toHaveAttribute('aria-hidden', 'true');
});

test('shows a badge as a chip after the note', () => {
  render(Section, { props: { title: 'Recently', accent: 'played', note: '3 games', badge: 'Stale', children } });

  expect(screen.getByRole('paragraph')).toHaveTextContent(/^3 games Stale$/);
  expect(screen.getByText('Stale')).toHaveClass('chip');
});

test('shows a badge without a note', () => {
  render(Section, { props: { title: 'Recently', accent: 'played', badge: 'New', children } });

  expect(screen.getByRole('paragraph')).toHaveTextContent(/^New$/);
});
