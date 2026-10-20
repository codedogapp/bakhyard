import { type AppId, appHref } from '../shell/apps';
import type { Item } from './api';

const MINUTE_MS = 60_000;
const HOUR_MS = 60 * MINUTE_MS;
const DAY_MS = 24 * HOUR_MS;
const MINUTES_PER_HOUR = 60;
const HOURS_PER_DAY = 24;

const relativeTime = new Intl.RelativeTimeFormat('en-GB', { numeric: 'always' });

// Halves round away from zero, so a time rounds the same way before and after now.
function nearest(ms: number, unitMs: number): number {
  return Math.sign(ms) * Math.round(Math.abs(ms) / unitMs);
}

/** When an item is due, from its RFC 3339 due time, relative to now: "now", "in 3 hours", "1 day ago". */
export function dueLabel(due: string, now: Date): string {
  const ms = Date.parse(due) - now.getTime();
  const minutes = nearest(ms, MINUTE_MS);
  if (minutes === 0) {
    return 'now';
  }
  if (Math.abs(minutes) < MINUTES_PER_HOUR) {
    return relativeTime.format(minutes, 'minute');
  }
  const hours = nearest(ms, HOUR_MS);
  if (Math.abs(hours) < HOURS_PER_DAY) {
    return relativeTime.format(hours, 'hour');
  }
  return relativeTime.format(nearest(ms, DAY_MS), 'day');
}

/** "item" or "items", to follow a count. */
export function itemNoun(count: number): string {
  return count === 1 ? 'item' : 'items';
}

/** The feed's note, such as "6 items · most urgent first"; none with no items. */
export function countNote(count: number): string | undefined {
  if (count === 0) {
    return undefined;
  }
  return `${count} ${itemNoun(count)} · most urgent first`;
}

/** How many of the items belong to the app. */
export function itemCount(items: readonly Pick<Item, 'app'>[], app: AppId): number {
  return items.filter((item) => item.app === app).length;
}

/** Where an item's row links to: its app's page, until items carry their own url (decision 50). */
export function itemHref(item: Pick<Item, 'app'>): string {
  return appHref(item.app);
}
