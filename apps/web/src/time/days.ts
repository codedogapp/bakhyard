import type { Day } from './report';
import { formatHours } from './week';

/** A day as the Time page lists it. */
export interface DayRow {
  /** The day, as YYYY-MM-DD. */
  date: string;
  /** Its date: "15 Oct". */
  label: string;
  /** Its weekday, "Thursday", or "Today". */
  title: string;
  /** The hours logged on it: "7h 30m", "0h". */
  hours: string;
  /** Whether any time is logged on it. */
  logged: boolean;
  missing: boolean;
}

// A report's date names a day, not a moment: read and written in UTC, it stays that day in every time zone.
function format(date: string, options: Intl.DateTimeFormatOptions): string {
  return new Date(date).toLocaleDateString('en-GB', { ...options, timeZone: 'UTC' });
}

/** A day by its weekday and date: "Monday 19 Oct". */
export function dayName(date: string): string {
  return format(date, { weekday: 'long', day: 'numeric', month: 'short' });
}

/** The report's days as the Time page lists them, in the report's order, whose last day is today (decision 181). */
export function dayRows(days: readonly Day[]): DayRow[] {
  return days.map((day, index) => ({
    date: day.date,
    label: format(day.date, { day: 'numeric', month: 'short' }),
    title: index === days.length - 1 ? 'Today' : format(day.date, { weekday: 'long' }),
    hours: formatHours(day.loggedHours),
    logged: day.loggedHours > 0,
    missing: day.missing,
  }));
}

/** How many of the days miss their hours: "1 day missing" or "2 days missing"; none when none does. */
export function missingNote(days: readonly Day[]): string | undefined {
  const count = days.filter((day) => day.missing).length;
  if (count === 0) {
    return undefined;
  }
  return count === 1 ? '1 day missing' : `${count} days missing`;
}
