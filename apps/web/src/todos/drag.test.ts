import { flushSync } from 'svelte';
import { afterEach, expect, test, vi } from 'vitest';

import type { Priority, Todo } from './api';
import { Drag } from './drag.svelte';
import { TodosState } from './state.svelte';

afterEach(() => {
  document.body.replaceChildren();
});

function todo(id: number, priority: Priority): Todo {
  return { id, title: `Todo ${id}`, done: false, priority };
}

/** The page's rows and grips, High 6, 4 and 2 from 100 px down, then Normal 5 from 400 px, each row 50 px tall. */
function page(editor = { busy: false }) {
  const list = new TodosState();
  list.todos = [todo(6, 'high'), todo(5, 'normal'), todo(4, 'high'), todo(2, 'high')];
  const drop = vi.spyOn(list, 'drop').mockResolvedValue();
  const drag = new Drag(list, editor);
  const tops = new Map([
    [6, 100],
    [4, 150],
    [2, 200],
    [5, 400],
  ]);
  const grips = new Map<number, HTMLElement>();
  const cleanups = new Map<number, () => void>();
  for (const [id, top] of tops) {
    const row = document.body.appendChild(document.createElement('li'));
    row.getBoundingClientRect = () => new DOMRect(0, top, 300, 50);
    const grip = row.appendChild(document.createElement('span'));
    grip.setPointerCapture = vi.fn<(pointerId: number) => void>();
    drag.row(id)(row);
    cleanups.set(id, drag.grip(id)(grip) as () => void);
    grips.set(id, grip);
  }
  flushSync();
  return { list, drop, drag, grip: (id: number) => grips.get(id) as HTMLElement, cleanups };
}

function pointer(type: string, clientY: number, init: PointerEventInit = {}): PointerEvent {
  return new PointerEvent(type, { pointerId: 1, isPrimary: true, button: 0, clientY, cancelable: true, ...init });
}

test('lifts the row under the pointer and marks where the todo would land', () => {
  const { drag, grip } = page();

  expect(drag.waiting).toBe(false);
  const down = pointer('pointerdown', 110);
  grip(6).dispatchEvent(down);

  expect(down.defaultPrevented, 'no text selected on the way').toBe(true);
  expect(grip(6).setPointerCapture).toHaveBeenCalledExactlyOnceWith(1);
  expect(drag.look(6)).toBe('lifted');
  expect(drag.look(4)).toBeUndefined();

  grip(6).dispatchEvent(pointer('pointermove', 190));

  expect(drag.look(6)).toBe('lifted');
  expect(drag.look(2)).toBe('drop-before');
  expect(drag.look(4)).toBeUndefined();

  grip(6).dispatchEvent(pointer('pointermove', 240));

  expect(drag.look(2)).toBe('drop-after');
  expect(drag.look(4)).toBeUndefined();
});

test('drops the todo where the pointer lets go, then lays the rows down', () => {
  const { drop, drag, grip } = page();

  grip(6).dispatchEvent(pointer('pointerdown', 110));
  grip(6).dispatchEvent(pointer('pointermove', 420));
  grip(6).dispatchEvent(pointer('pointerup', 420));

  expect(drop).toHaveBeenCalledExactlyOnceWith(6, { priority: 'normal', place: { before: 5 } });
  expect(drag.look(6)).toBeUndefined();
  expect(drag.look(5)).toBeUndefined();
});

test.each([
  { name: 'next to its own place', y: 160 },
  { name: 'without a move', y: undefined },
])('drops nothing when the pointer lets go $name', ({ y }) => {
  const { drop, drag, grip } = page();

  grip(4).dispatchEvent(pointer('pointerdown', 160));
  if (y !== undefined) {
    grip(4).dispatchEvent(pointer('pointermove', y));
  }
  grip(4).dispatchEvent(pointer('pointerup', y ?? 160));

  expect(drop).not.toHaveBeenCalled();
  expect(drag.look(4)).toBeUndefined();
});

test.each([
  { name: 'a pointer that is not the first', init: { isPrimary: false } },
  { name: 'a button that is not the main one', init: { button: 2 } },
])('lifts nothing for $name', ({ init }) => {
  const { drag, grip } = page();
  const down = pointer('pointerdown', 110, init);

  grip(6).dispatchEvent(down);

  expect(down.defaultPrevented).toBe(false);
  expect(drag.look(6)).toBeUndefined();
  expect(grip(6).setPointerCapture).not.toHaveBeenCalled();
});

test.each([
  { name: 'an edit', busy: true, dropping: false },
  { name: 'a drop', busy: false, dropping: true },
])('lifts nothing while $name waits for the service', ({ busy, dropping }) => {
  const { list, drag, grip } = page({ busy });
  list.dropping = dropping;

  grip(6).dispatchEvent(pointer('pointerdown', 110));

  expect(drag.waiting).toBe(true);
  expect(drag.look(6)).toBeUndefined();
});

test('drags one todo at a time, by the pointer that lifted it', () => {
  const { drop, drag, grip } = page();

  grip(6).dispatchEvent(pointer('pointerdown', 110));
  grip(2).dispatchEvent(pointer('pointerdown', 210, { pointerId: 2 }));
  grip(6).dispatchEvent(pointer('pointermove', 420, { pointerId: 2 }));
  grip(6).dispatchEvent(pointer('pointerup', 420, { pointerId: 2 }));

  expect(drag.look(6)).toBe('lifted');
  expect(drag.look(2)).toBeUndefined();
  expect(grip(2).setPointerCapture).not.toHaveBeenCalled();
  expect(drag.look(5)).toBeUndefined();
  expect(drop).not.toHaveBeenCalled();
});

test('drops nothing when the browser takes the pointer away', () => {
  const { drop, drag, grip } = page();

  grip(6).dispatchEvent(pointer('pointerdown', 110));
  grip(6).dispatchEvent(pointer('pointermove', 420));
  grip(6).dispatchEvent(new PointerEvent('lostpointercapture', { pointerId: 1 }));
  grip(6).dispatchEvent(pointer('pointerup', 420));

  expect(drop).not.toHaveBeenCalled();
  expect(drag.look(6)).toBeUndefined();
});

test('ends the drag when its grip goes, and only then', () => {
  const { drop, drag, grip, cleanups } = page();

  grip(6).dispatchEvent(pointer('pointerdown', 110));
  cleanups.get(4)?.();
  expect(drag.look(6)).toBe('lifted');

  cleanups.get(6)?.();
  grip(6).dispatchEvent(pointer('pointerdown', 110));

  expect(drag.look(6)).toBeUndefined();
  expect(drop).not.toHaveBeenCalled();
});

test("keeps a todo's newer row when its older one goes, and measures no row that has gone", () => {
  const { drop, drag, grip } = page();
  const older = document.createElement('li');
  const newer = document.createElement('li');
  newer.getBoundingClientRect = () => new DOMRect(0, 700, 300, 50);
  const removeOlder = drag.row(5)(older) as () => void;
  const removeNewer = drag.row(5)(newer) as () => void;
  removeOlder();

  const dragTo = (y: number) => {
    grip(6).dispatchEvent(pointer('pointerdown', 110));
    grip(6).dispatchEvent(pointer('pointermove', y));
    grip(6).dispatchEvent(pointer('pointerup', y));
  };

  dragTo(690);
  removeNewer();
  dragTo(690);

  expect(drop.mock.calls).toEqual([
    [6, { priority: 'normal', place: { before: 5 } }],
    [6, { priority: 'high', place: { after: 2 } }],
  ]);
});
