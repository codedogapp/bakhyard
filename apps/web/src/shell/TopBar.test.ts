import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import type { Connection } from '../home/poll';
import TopBar from './TopBar.svelte';

const now = new Date('2026-10-20T01:11:59Z');

test('shows the local time and city', () => {
  render(TopBar, { props: { now, connection: 'online' } });

  expect(screen.getByText('10:11 · Tokyo')).toBeInTheDocument();
});

test.each<{ connection: Connection; asOf?: Date; unavailable?: number; want: string }>([
  { connection: 'connecting', want: '' },
  { connection: 'online', want: 'Online' },
  { connection: 'online', unavailable: 2, want: 'Online · 2 sources unavailable' },
  { connection: 'offline', asOf: new Date('2026-10-20T01:10:00Z'), want: 'Offline · as of 10:10' },
])('says "$want" in its status when $connection', ({ connection, asOf, unavailable, want }) => {
  render(TopBar, { props: { now, connection, asOf, unavailable } });

  expect(screen.getByRole('status').textContent).toBe(want);
});
