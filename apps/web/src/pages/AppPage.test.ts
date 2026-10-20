import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import type { App } from '../shell/apps';
import AppPage from './AppPage.svelte';

const pulls: App = {
  id: 'pulls',
  label: 'PRs',
  name: 'Pull requests',
  description: 'Bitbucket PRs and reviewer nudges',
  number: '03',
};

test('names the app, says what it does, and that its page is coming soon', () => {
  render(AppPage, { props: { app: pulls } });

  expect(screen.getByRole('heading', { level: 1, name: 'Pull requests' })).toBeInTheDocument();
  expect(screen.getAllByRole('paragraph').map((p) => p.textContent)).toEqual([
    'Coming soon',
    'Bitbucket PRs and reviewer nudges',
  ]);
});
