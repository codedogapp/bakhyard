import type { Home } from './api';

/** How many sources home couldn't reach for the Home: none without a Home, or with one that names no sources. */
export function unavailableSources(home?: Home): number {
  return home?.sources?.filter(({ state }) => state === 'unavailable').length ?? 0;
}
