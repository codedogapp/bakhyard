import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import App from './App.svelte';

test('shows the Bakhyard heading', () => {
  render(App);

  expect(screen.getByRole('heading', { level: 1, name: 'Bakhyard' })).toBeInTheDocument();
});
