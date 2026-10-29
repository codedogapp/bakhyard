import { fireEvent, render, screen, within } from '@testing-library/svelte';
import { beforeEach, expect, test, vi } from 'vitest';

import Bell from './Bell.svelte';
import { type Push, preparePush } from './push';

vi.mock('./push', () => ({ preparePush: vi.fn() }));

const prepare = vi.mocked(preparePush);

beforeEach(() => {
  prepare.mockReset();
});

function panelOf(bell: HTMLElement): HTMLElement {
  const panel = document.getElementById(bell.getAttribute('popovertarget') ?? '');
  if (!panel) {
    throw new Error('the bell opens no panel');
  }
  return panel;
}

// jsdom can't open a popover and hides its content, so the tests look inside the closed panel, and fire the event
// that a browser fires as the panel opens or closes.
function toggled(newState: 'open' | 'closed'): Event {
  return Object.assign(new Event('toggle'), { newState });
}

// Lets the bell finish what it does after a check, so a test can see that it did nothing.
function settle(): Promise<unknown> {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

test.each<{ push: Push; name: string; slashed: boolean }>([
  { push: { kind: 'on' }, name: 'Notifications: On', slashed: false },
  { push: { kind: 'off', enable: vi.fn() }, name: 'Notifications: Off', slashed: true },
  { push: { kind: 'blocked' }, name: 'Notifications: Blocked', slashed: true },
  { push: { kind: 'unsupported' }, name: 'Notifications: Unavailable', slashed: true },
  { push: { kind: 'failed', reason: 'offline' }, name: 'Notifications: Failed', slashed: true },
])('is named "$name" once it has checked, and slashed: $slashed', async ({ push, name, slashed }) => {
  prepare.mockResolvedValue(push);
  render(Bell);

  const bell = await screen.findByRole('button', { name });
  expect(bell.querySelectorAll('path')).toHaveLength(slashed ? 3 : 1);
  // The slash cuts its gap out of the bell, which is the last path.
  const gap = bell.querySelector('mask');
  expect(bell.querySelector('svg > path:last-child')?.getAttribute('mask') ?? null).toBe(
    gap ? `url(#${gap.id})` : null
  );
  expect(gap !== null).toBe(slashed);
});

test('is a plain bell named "Notifications" while it checks', () => {
  prepare.mockReturnValue(new Promise(() => {}));
  render(Bell);

  const bell = screen.getByRole('button', { name: 'Notifications' });
  expect(bell.querySelectorAll('path')).toHaveLength(1);
  expect(prepare).toHaveBeenCalledOnce();
});

test('opens a panel that says what it knows', async () => {
  prepare.mockResolvedValue({ kind: 'blocked' });
  render(Bell);

  const panel = panelOf(await screen.findByRole('button', { name: 'Notifications: Blocked' }));
  expect(panel).toHaveAttribute('popover', 'auto');
  expect(within(panel).getByRole('status', { hidden: true }).textContent).toBe(
    'Notifications are blocked in the browser settings'
  );
});

test('checks again as its panel opens, and says what it knew until then', async () => {
  prepare.mockResolvedValueOnce({ kind: 'on' });
  render(Bell);
  const bell = await screen.findByRole('button', { name: 'Notifications: On' });
  const again = Promise.withResolvers<Push>();
  prepare.mockReturnValueOnce(again.promise);

  await fireEvent(panelOf(bell), toggled('open'));

  expect(prepare).toHaveBeenCalledTimes(2);
  expect(bell).toHaveAccessibleName('Notifications: On');
  again.resolve({ kind: 'blocked' });
  await vi.waitFor(() => expect(bell).toHaveAccessibleName('Notifications: Blocked'));
});

test("doesn't check as its panel closes", async () => {
  prepare.mockResolvedValue({ kind: 'on' });
  render(Bell);
  const bell = await screen.findByRole('button', { name: 'Notifications: On' });

  await fireEvent(panelOf(bell), toggled('closed'));

  expect(prepare).toHaveBeenCalledOnce();
});

test('keeps the latest check, when an earlier one ends after it', async () => {
  const first = Promise.withResolvers<Push>();
  prepare.mockReturnValueOnce(first.promise).mockResolvedValueOnce({ kind: 'on' });
  render(Bell);
  const bell = screen.getByRole('button', { name: 'Notifications' });

  await fireEvent(panelOf(bell), toggled('open'));
  await vi.waitFor(() => expect(bell).toHaveAccessibleName('Notifications: On'));
  first.resolve({ kind: 'blocked' });
  await settle();

  expect(bell).toHaveAccessibleName('Notifications: On');
});

test('turns notifications on from its panel, then says so, and takes the focus the button had', async () => {
  const on = Promise.withResolvers<Push>();
  prepare.mockResolvedValueOnce({ kind: 'off', enable: async () => undefined }).mockReturnValueOnce(on.promise);
  render(Bell);
  const button = await screen.findByRole('button', { name: 'Turn on notifications', hidden: true });
  // jsdom focuses nothing in a closed popover, so the panel is shown as an open one would be.
  panelOf(screen.getByRole('button', { name: 'Notifications: Off' })).style.display = 'block';
  button.focus();
  expect(button).toHaveFocus();

  await fireEvent.click(button);
  // The check after the change answers late, and the focus waits for it.
  await settle();
  on.resolve({ kind: 'on' });

  const bell = await screen.findByRole('button', { name: 'Notifications: On' });
  expect(screen.getByRole('status', { hidden: true }).textContent).toBe('Notifications on');
  await vi.waitFor(() => expect(bell).toHaveFocus());
});

test('leaves the focus alone when it went elsewhere while notifications turned on', async () => {
  const on = Promise.withResolvers<Push>();
  prepare.mockResolvedValueOnce({ kind: 'off', enable: async () => undefined }).mockReturnValueOnce(on.promise);
  const { container } = render(Bell);
  const elsewhere = container.appendChild(document.createElement('input'));

  await fireEvent.click(await screen.findByRole('button', { name: 'Turn on notifications', hidden: true }));
  elsewhere.focus();
  on.resolve({ kind: 'on' });
  await screen.findByRole('button', { name: 'Notifications: On' });
  await settle();

  expect(elsewhere).toHaveFocus();
});
