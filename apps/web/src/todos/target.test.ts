import { expect, test } from 'vitest';

import { dropTarget, type GroupBox } from './target';

// High 6, 4 and 2 from 100 to 250 px; Normal 5 and 3 from 400 to 500; Low 1 from 700 to 750. Each row is 50 px tall.
const groups: GroupBox[] = [
  {
    priority: 'high',
    rows: [
      { id: 6, top: 100, bottom: 150 },
      { id: 4, top: 150, bottom: 200 },
      { id: 2, top: 200, bottom: 250 },
    ],
  },
  {
    priority: 'normal',
    rows: [
      { id: 5, top: 400, bottom: 450 },
      { id: 3, top: 450, bottom: 500 },
    ],
  },
  { priority: 'low', rows: [{ id: 1, top: 700, bottom: 750 }] },
];

test.each([
  { id: 6, y: 180, want: { priority: 'high', place: { before: 2 } } },
  { id: 6, y: 225, want: { priority: 'high', place: { before: 2 } } },
  { id: 6, y: 230, want: { priority: 'high', place: { after: 2 } } },
  { id: 6, y: 300, want: { priority: 'high', place: { after: 2 } } },
  { id: 2, y: 120, want: { priority: 'high', place: { before: 6 } } },
  { id: 2, y: 140, want: { priority: 'high', place: { before: 4 } } },
  { id: 4, y: 0, want: { priority: 'high', place: { before: 6 } } },
  { id: 6, y: 420, want: { priority: 'normal', place: { before: 5 } } },
  { id: 6, y: 440, want: { priority: 'normal', place: { before: 3 } } },
  { id: 6, y: 480, want: { priority: 'normal', place: { after: 3 } } },
  { id: 2, y: 340, want: { priority: 'normal', place: { before: 5 } } },
  { id: 6, y: 600, want: { priority: 'normal', place: { after: 3 } } },
  { id: 6, y: 650, want: { priority: 'low', place: { before: 1 } } },
  { id: 6, y: 900, want: { priority: 'low', place: { after: 1 } } },
  { id: 1, y: 160, want: { priority: 'high', place: { before: 4 } } },
])("drops todo $id at $y px in the nearest group, by its rows' middles", ({ id, y, want }) => {
  expect(dropTarget(groups, id, y)).toEqual(want);
});

test.each([
  { id: 6, y: 0 },
  { id: 6, y: 110 },
  { id: 6, y: 160 },
  { id: 4, y: 140 },
  { id: 4, y: 210 },
  { id: 2, y: 260 },
  { id: 2, y: 320 },
  { id: 1, y: 650 },
  { id: 1, y: 900 },
])('leaves todo $id where it is when dropped at $y px, next to its own place', ({ id, y }) => {
  expect(dropTarget(groups, id, y)).toBeUndefined();
});

test.each([
  { name: 'no groups', groups: [] },
  { name: 'a group without rows', groups: [{ priority: 'high', rows: [] }] },
] as const)('drops nowhere with $name', ({ groups }) => {
  expect(dropTarget(groups, 6, 100)).toBeUndefined();
});

test('passes over a group none of whose rows it could measure', () => {
  const measured: GroupBox[] = [{ priority: 'high', rows: [] }, groups[1]];

  expect(dropTarget(measured, 6, 0)).toEqual({ priority: 'normal', place: { before: 5 } });
});
