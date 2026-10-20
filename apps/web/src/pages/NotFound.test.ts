import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import NotFound from './NotFound.svelte';

test('says there is no page at the path', () => {
  render(NotFound, { props: { path: '/elsewhere' } });

  expect(screen.getByRole('heading', { level: 1, name: 'Page not found' })).toBeInTheDocument();
  expect(screen.getByRole('paragraph')).toHaveTextContent("There's no page at /elsewhere.");
});
