import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import type { Connection } from '../home/poll';
import TopBar from './TopBar.svelte';

const now = new Date('2026-10-20T01:11:59Z');

test('shows the local time and city', () => {
  render(TopBar, { props: { now, connection: 'online' } });

  expect(screen.getByText('10:11 · Tokyo')).toBeInTheDocument();
});

test.each<{ connection: Connection; asOf?: Date; want: string }>([
  { connection: 'connecting', want: '' },
  { connection: 'online', want: 'Online' },
  { connection: 'offline', asOf: new Date('2026-10-20T01:10:00Z'), want: 'Offline · as of 10:10' },
])('says "$want" in its status when $connection', ({ connection, asOf, want }) => {
  render(TopBar, { props: { now, connection, asOf } });

  expect(screen.getByRole('status')).toHaveTextContent(want, { normalizeWhitespace: true });
});
