import { expect, test } from 'vitest';

import type { Home } from './api';
import { unavailableSources } from './sources';

const home: Home = { name: 'Ada', items: [] };

test.each<{ name: string; home?: Home; want: number }>([
  { name: 'no Home', want: 0 },
  { name: 'a Home without sources', home, want: 0 },
  { name: 'no sources', home: { ...home, sources: [] }, want: 0 },
  { name: 'every source reached', home: { ...home, sources: [{ state: 'ok' }, { state: 'ok' }] }, want: 0 },
  {
    name: 'some sources unreached',
    home: { ...home, sources: [{ state: 'unavailable' }, { state: 'ok' }, { state: 'unavailable' }] },
    want: 2,
  },
])('counts $want unavailable with $name', ({ home, want }) => {
  expect(unavailableSources(home)).toBe(want);
});
