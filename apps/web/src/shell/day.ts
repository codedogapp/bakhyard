const dayMs = 86_400_000;
const dateFormat = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short' });

// The date's day in the device's time zone, as a count of days, so a difference of two is whole even across summer time.
function dayNumber(date: Date): number {
  return Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()) / dayMs;
}

/** The day of a time, in the device's time zone, as a list row's label: "Today", "Yesterday" or "19 Oct". */
export function dayLabel(time: string, now: Date): string {
  const day = new Date(time);
  const daysAgo = dayNumber(now) - dayNumber(day);
  if (daysAgo === 0) {
    return 'Today';
  }
  return daysAgo === 1 ? 'Yesterday' : dateFormat.format(day);
}
