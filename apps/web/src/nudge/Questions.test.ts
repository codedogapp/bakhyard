import { render, screen } from '@testing-library/svelte';
import { expect, test, vi } from 'vitest';

import type { Question } from './nudge';
import Questions from './Questions.svelte';

const ask = (id: string, text: string): Question => ({
  id,
  project: 'atlas',
  text,
  details: '',
  options: [],
  asked_at: '2026-10-21T00:41:00Z',
});

test('lists the questions that wait, and counts them', () => {
  render(Questions, { questions: [ask('1', 'Merge?'), ask('2', 'Ship it?')], answered: vi.fn() });

  expect(screen.getByText('2 waiting')).toBeInTheDocument();
  expect(screen.getAllByRole('heading', { level: 3 }).map((heading) => heading.textContent)).toEqual([
    'Merge?',
    'Ship it?',
  ]);
});

test('says when nothing waits', () => {
  render(Questions, { questions: [], answered: vi.fn() });

  expect(screen.getByText('No agent is waiting for you.')).toBeInTheDocument();
  expect(screen.queryByText(/waiting$/)).not.toBeInTheDocument();
});
