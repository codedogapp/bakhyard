import { render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import App from './App.svelte';

beforeEach(() => {
  vi.stubGlobal(
    'fetch',
    vi.fn<typeof fetch>(async () => Response.json({ name: 'Ada', items: [{ late: true }] }))
  );
});

afterEach(() => {
  vi.unstubAllGlobals();
});

test('shows the frame, then the hero and the status once the Home loads', async () => {
  render(App);

  expect(screen.getByText(/ · Tokyo$/)).toBeInTheDocument();
  expect(screen.getByRole('banner')).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Home', current: 'page' })).toBeInTheDocument();
  expect(await screen.findByRole('heading', { level: 1, name: /^Good \w+, Ada\.$/ })).toBeInTheDocument();
  expect(screen.getByText('1 thing needs you today. 1 is late.')).toBeInTheDocument();
  expect(screen.getByRole('status')).toHaveTextContent('Online');
});
