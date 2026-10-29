import { fireEvent, render, screen } from '@testing-library/svelte';
import { expect, test, vi } from 'vitest';

import Notifications from './Notifications.svelte';
import type { Push } from './push';

test.each<{ push?: Push; want: string }>([
  { push: { kind: 'on' }, want: 'Notifications on' },
  { push: { kind: 'blocked' }, want: 'Notifications are blocked in the browser settings' },
  { push: { kind: 'unsupported' }, want: 'Notifications need the Home Screen app' },
  { push: { kind: 'failed', reason: 'offline' }, want: 'Notifications failed: offline' },
  { push: undefined, want: 'Checking notifications…' },
])('says "$want", with no button', ({ push, want }) => {
  render(Notifications, { push, changed: vi.fn() });

  expect(screen.getByRole('status').textContent).toBe(want);
  expect(screen.queryByRole('button')).not.toBeInTheDocument();
});

test('offers only its button when notifications are off', () => {
  render(Notifications, { push: { kind: 'off', enable: vi.fn() }, changed: vi.fn() });

  expect(screen.getByRole('button', { name: 'Turn on notifications' })).toBeInTheDocument();
  expect(screen.getByRole('status').textContent).toBe('');
});

test('turns notifications on with its button, and says that it did', async () => {
  const enable = vi.fn(async () => undefined);
  const changed = vi.fn();
  render(Notifications, { push: { kind: 'off', enable }, changed });

  await fireEvent.click(screen.getByRole('button', { name: 'Turn on notifications' }));

  expect(enable).toHaveBeenCalledOnce();
  expect(changed).toHaveBeenCalledOnce();
});

test('turns notifications on once, however often the button is tapped while it works', async () => {
  const { promise, resolve } = Promise.withResolvers<string | undefined>();
  const enable = vi.fn(() => promise);
  const changed = vi.fn();
  render(Notifications, { push: { kind: 'off', enable }, changed });
  const button = screen.getByRole('button', { name: 'Turn on notifications' });

  await fireEvent.click(button);
  expect(button).toHaveAttribute('aria-disabled', 'true');
  await fireEvent.click(button);
  resolve(undefined);

  await vi.waitFor(() => expect(changed).toHaveBeenCalledOnce());
  expect(enable).toHaveBeenCalledOnce();
  expect(button).toHaveAttribute('aria-disabled', 'false');
});

test('says why turning notifications on failed, and keeps the button', async () => {
  const changed = vi.fn();
  render(Notifications, { push: { kind: 'off', enable: async () => 'denied' }, changed });

  await fireEvent.click(screen.getByRole('button', { name: 'Turn on notifications' }));

  expect(await screen.findByText('Notifications failed: denied')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Turn on notifications' })).toBeInTheDocument();
  expect(changed).not.toHaveBeenCalled();
});

test('forgets a failure once the bell checks again', async () => {
  const { rerender } = render(Notifications, {
    push: { kind: 'off', enable: async () => 'dismissed' },
    changed: vi.fn(),
  });
  await fireEvent.click(screen.getByRole('button', { name: 'Turn on notifications' }));
  await screen.findByText('Notifications failed: dismissed');

  await rerender({ push: { kind: 'off', enable: vi.fn() } });

  expect(screen.getByRole('status').textContent).toBe('');
});
