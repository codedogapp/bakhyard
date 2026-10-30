import { expect, test } from 'vitest';

import { titleLabel } from './labels';

const words = (count: number) => Array.from({ length: count }, (_, index) => `word${index}`).join(' ');

test.each([
  { name: 'a short title as it is', title: 'Back up the home lab', want: 'Back up the home lab' },
  { name: 'a blank title as untitled', title: '', want: 'Untitled plan' },
  { name: 'a title of spaces alone as untitled', title: ' \n\t ', want: 'Untitled plan' },
  {
    name: 'a prompt of many lines on one',
    title: ' Plan a move:\n\n- the blog\n- the backups ',
    want: 'Plan a move: - the blog - the backups',
  },
  { name: 'a title of 80 characters whole', title: 'a'.repeat(80), want: 'a'.repeat(80) },
  { name: 'a longer one whole up to a word that ends at 80', title: words(20), want: `${words(13)}…` },
  { name: 'a longer one cut at its last space', title: `x${words(20)}`, want: `x${words(12)}…` },
  {
    name: 'a long word cut at 80 characters',
    title: `${'a'.repeat(30)} ${'b'.repeat(60)}`,
    want: `${'a'.repeat(30)} ${'b'.repeat(49)}…`,
  },
  { name: 'a long one counted in characters', title: '🙂'.repeat(81), want: `${'🙂'.repeat(80)}…` },
])('shows $name', ({ title, want }) => {
  expect(titleLabel(title)).toBe(want);
});
