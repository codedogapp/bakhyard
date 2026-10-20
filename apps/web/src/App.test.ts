import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import App from './App.svelte';

test('shows the top bar, the header and the Bakhyard heading', () => {
  render(App);

  expect(screen.getByText(/ · Tokyo$/)).toBeInTheDocument();
  expect(screen.getByRole('banner')).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Home', current: 'page' })).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 1, name: 'Bakhyard' })).toBeInTheDocument();
});
