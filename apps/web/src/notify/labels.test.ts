import { expect, test } from 'vitest';

import { bellName, pushStatus } from './labels';
import type { Push } from './push';

test.each<{ push?: Push; name: string; status: string }>([
  { push: undefined, name: 'Notifications', status: 'Checking notifications…' },
  { push: { kind: 'on' }, name: 'Notifications: On', status: 'Notifications on' },
  { push: { kind: 'off', enable: async () => undefined }, name: 'Notifications: Off', status: '' },
  {
    push: { kind: 'blocked' },
    name: 'Notifications: Blocked',
    status: 'Notifications are blocked in the browser settings',
  },
  {
    push: { kind: 'unsupported' },
    name: 'Notifications: Unavailable',
    status: 'Notifications need the Home Screen app',
  },
  {
    push: { kind: 'failed', reason: 'offline' },
    name: 'Notifications: Failed',
    status: 'Notifications failed: offline',
  },
])('names the bell "$name" and says "$status"', ({ push, name, status }) => {
  expect(bellName(push)).toBe(name);
  expect(pushStatus(push)).toBe(status);
});
