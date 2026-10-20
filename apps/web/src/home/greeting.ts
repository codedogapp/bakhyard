import type { Item } from './api';

const MORNING_HOUR = 5;
const AFTERNOON_HOUR = 12;
const EVENING_HOUR = 18;

/** The hero's date, such as "Tuesday 20 October", in the local time zone. */
export function dateLabel(now: Date): string {
  return new Intl.DateTimeFormat('en-GB', { weekday: 'long', day: 'numeric', month: 'long' }).format(now);
}

/** "Good morning" from 05:00, "Good afternoon" from 12:00, and "Good evening" from 18:00 until 05:00. */
export function greeting(now: Date): string {
  const hour = now.getHours();
  if (hour < MORNING_HOUR) {
    return 'Good evening';
  }
  if (hour < AFTERNOON_HOUR) {
    return 'Good morning';
  }
  if (hour < EVENING_HOUR) {
    return 'Good afternoon';
  }
  return 'Good evening';
}

/** The hero's summary, such as "6 things need you today. 2 are late." */
export function summary(items: readonly Item[]): string {
  if (items.length === 0) {
    return 'All clear today.';
  }
  const needs = items.length === 1 ? '1 thing needs you today.' : `${items.length} things need you today.`;
  const late = items.filter((item) => item.late).length;
  if (late === 0) {
    return needs;
  }
  return `${needs} ${late === 1 ? '1 is' : `${late} are`} late.`;
}
