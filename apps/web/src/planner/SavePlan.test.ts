import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import SavePlan from './SavePlan.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

function renderButton() {
  render(SavePlan, { props: { id: 'p1' } });
  return screen.getByRole('button', { name: 'Save to Obsidian' });
}

test('saves the plan in the vault, and says as which note', async () => {
  const saved = Response.json({ path: 'Plans/2026-10-30-trip.md' }, { status: 201 });
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(saved);
  vi.stubGlobal('fetch', fetchMock);
  const button = renderButton();
  expect(screen.getByRole('status').textContent).toBe('');

  await fireEvent.click(button);

  await vi.waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('Saved as “Plans/2026-10-30-trip”'));
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/planner/plans/p1/note', { method: 'POST' });
  expect(button).toHaveAttribute('aria-disabled', 'false');
});

test('takes no click while it saves, and says why the save failed', async () => {
  const answer = Promise.withResolvers<Response>();
  const fetchMock = vi.fn<typeof fetch>().mockReturnValue(answer.promise);
  vi.stubGlobal('fetch', fetchMock);
  const button = renderButton();

  await fireEvent.click(button);
  await fireEvent.click(button);

  expect(screen.getByRole('status')).toHaveTextContent('Saving…');
  expect(button).toHaveAttribute('aria-disabled', 'true');

  answer.resolve(new Response('Bad Gateway\n', { status: 502 }));

  await vi.waitFor(() => expect(screen.getByRole('status')).toHaveTextContent("Couldn't save the plan in the vault."));
  expect(button).toHaveAttribute('aria-disabled', 'false');
  expect(fetchMock).toHaveBeenCalledOnce();
});
