import { loadSnapshot, type Polled, type Polls } from '../shell/snapshot';
import type { Week } from './week';

/** A day of the time report. */
export interface Day {
  /** The day in the time service's zone, as YYYY-MM-DD. */
  date: string;
  loggedHours: number;
  /** Whether it's a weekday before today with no hours logged. */
  missing: boolean;
}

/** The body of GET /api/time/report: the time service's report of the last 7 days (decisions 181 and 182). */
export interface Report extends Polls {
  /** The last 7 days, the earliest first and today last; none before the first poll that succeeded. */
  days?: Day[];
  /** This week's hours against the target; none before the first poll that succeeded. */
  week?: Week;
}

/** The time report as the Time page has it: loading, off when Jira isn't set up, failed to load, or loaded. */
export type TimeReport = Polled<Report>;

/** Loads the time report; never fails, since a failed load is one of the outcomes. */
export function loadReport(): Promise<TimeReport> {
  return loadSnapshot('/api/time/report');
}
