import { afterEach, expect, test, vi } from 'vitest';

import { SaveState } from './save.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

function answer(response: Promise<Response>) {
  const fetchMock = vi.fn<typeof fetch>().mockReturnValue(response);
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const saved = (path: string) => Promise.resolve(Response.json({ path }, { status: 201 }));

test("saves the plan's plan, and says where", async () => {
  const fetchMock = answer(saved('Plans/2026-10-30-trip.md'));
  const saving = new SaveState();
  expect(saving.status).toEqual({ kind: 'idle' });

  await saving.save('p1');

  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/planner/plans/p1/note', { method: 'POST' });
  expect(saving.status).toEqual({ kind: 'saved', path: 'Plans/2026-10-30-trip.md' });
});

test('saves nothing more while a save runs', async () => {
  const response = Promise.withResolvers<Response>();
  const fetchMock = answer(response.promise);
  const saving = new SaveState();

  const first = saving.save('p1');
  expect(saving.busy).toBe(true);
  expect(saving.status).toEqual({ kind: 'saving' });
  await saving.save('p1');
  response.resolve(await saved('Plans/2026-10-30-trip.md'));
  await first;

  expect(fetchMock).toHaveBeenCalledOnce();
  expect(saving.busy).toBe(false);
});

test('says why a save failed, and saves again, as a new note, when asked', async () => {
  const fetchMock = answer(Promise.resolve(new Response('Bad Gateway\n', { status: 502 })));
  const saving = new SaveState();

  await saving.save('p1');
  expect(saving.status).toEqual({ kind: 'failed', reason: "Couldn't save the plan in the vault." });
  fetchMock.mockReturnValue(saved('Plans/2026-10-30-trip.md'));
  await saving.save('p1');

  expect(saving.status).toEqual({ kind: 'saved', path: 'Plans/2026-10-30-trip.md' });
  expect(fetchMock).toHaveBeenCalledTimes(2);
});
