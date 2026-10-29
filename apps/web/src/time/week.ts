/** The hours logged this week, against the weekly target. */
export interface Week {
  loggedHours: number;
  targetHours: number;
}

const minutesPerHour = 60;
const full = 100;

// Narrow hours and minutes, as Jira writes them: "31h 30m", "40h". Hours always show, so zero reads "0h" rather than "".
const duration = new Intl.DurationFormat('en-GB', { style: 'narrow', hoursDisplay: 'always' });

function formatHours(hours: number): string {
  const minutes = Math.round(hours * minutesPerHour);
  return duration.format({ hours: Math.floor(minutes / minutesPerHour), minutes: minutes % minutesPerHour });
}

/** The hours logged against the target, each rounded to the minute: "31h 30m / 40h". */
export function hoursLabel(week: Week): string {
  return `${formatHours(week.loggedHours)} / ${formatHours(week.targetHours)}`;
}

/** How full the week's bar is, in percent: up to 100 once the target is met, and 0 without a target. */
export function weekPercent(week: Week): number {
  if (week.targetHours <= 0) {
    return 0;
  }
  return Math.min(week.loggedHours / week.targetHours, 1) * full;
}
