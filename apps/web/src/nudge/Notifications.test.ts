import { fireEvent, render, screen } from '@testing-library/svelte';
import { expect, test, vi } from 'vitest';

import Notifications from './Notifications.svelte';
import type { Push } from './push';

test.each<{ push?: Push; want: string }>([
  { push: { kind: 'on' }, want: 'Notifications on' },
  { push: { kind: 'blocked' }, want: 'Notifications are blocked in the browser settings' },
  { push: { kind: 'unsupported' }, want: 'Notifications need the Home Screen app' },
  { push: { kind: 'failed', reason: 'offline' }, want: 'Notifications failed: offline' },
  { push: undefined, want: '' },
])('says $push?.kind', ({ push, want }) => {
  render(Notifications, { push, changed: vi.fn() });

  expect(screen.getByRole('status')).toHaveTextContent(want);
  expect(screen.queryByRole('button')).not.toBeInTheDocument();
});

test('turns notifications on with its button, and says that it did', async () => {
  const enable = vi.fn(async () => undefined);
  const changed = vi.fn();
  render(Notifications, { push: { kind: 'off', enable }, changed });

  await fireEvent.click(screen.getByRole('button', { name: 'Turn on notifications' }));

  expect(enable).toHaveBeenCalledOnce();
  expect(changed).toHaveBeenCalledOnce();
});

test('says why turning notifications on failed, and keeps the button', async () => {
  const changed = vi.fn();
  render(Notifications, { push: { kind: 'off', enable: async () => 'denied' }, changed });

  await fireEvent.click(screen.getByRole('button', { name: 'Turn on notifications' }));

  expect(await screen.findByText('Notifications failed: denied')).toBeInTheDocument();
  expect(changed).not.toHaveBeenCalled();
});
