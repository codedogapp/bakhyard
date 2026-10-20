import { expect, test } from 'vitest';

import type { Connection } from '../home/poll';
import { statusLabel } from './status';

test.each<{ connection: Connection; asOf?: string; want: string }>([
  { connection: 'connecting', want: '' },
  { connection: 'online', asOf: '2026-10-20T01:10:00Z', want: 'Online' },
  { connection: 'offline', asOf: '2026-10-20T01:10:00Z', want: 'Offline · as of 10:10' },
  { connection: 'offline', want: 'Offline' },
])('says "$want" when $connection', ({ connection, asOf, want }) => {
  expect(statusLabel(connection, asOf ? new Date(asOf) : undefined)).toBe(want);
});
