import { expect, test } from 'vitest';

import type { Connection } from '../home/poll';
import { statusLabel } from './status';

test.each<{ connection: Connection; asOf?: string; unavailable?: number; want: string }>([
  { connection: 'connecting', want: '' },
  { connection: 'connecting', unavailable: 1, want: '' },
  { connection: 'online', asOf: '2026-10-20T01:10:00Z', want: 'Online' },
  { connection: 'online', unavailable: 0, want: 'Online' },
  { connection: 'online', unavailable: 1, want: 'Online · 1 source unavailable' },
  { connection: 'online', unavailable: 2, want: 'Online · 2 sources unavailable' },
  { connection: 'offline', asOf: '2026-10-20T01:10:00Z', want: 'Offline · as of 10:10' },
  { connection: 'offline', asOf: '2026-10-20T01:10:00Z', unavailable: 2, want: 'Offline · as of 10:10' },
  { connection: 'offline', want: 'Offline' },
  { connection: 'signed-out', want: 'Signed out' },
  { connection: 'signed-out', unavailable: 1, want: 'Signed out' },
])('says "$want" when $connection with $unavailable unavailable', ({ connection, asOf, unavailable, want }) => {
  expect(statusLabel(connection, asOf ? new Date(asOf) : undefined, unavailable)).toBe(want);
});
