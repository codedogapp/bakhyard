import type { Ticket } from '../jira/tickets';
import { dayName } from './days';
import type { Day } from './report';
import { formatHours } from './week';

/** One of a select's options: the value it picks, and its text. */
export interface Choice<T> {
  value: T;
  label: string;
}

const minutesPerHour = 60;
const quarter = 15;
const most = 12 * minutesPerHour;

/** The minutes the form starts on: a working day's 8 hours. */
export const workday = 8 * minutesPerHour;

/** The days the form offers: the report's, in its order, the earliest first: "Thursday 15 Oct", and "Today" last. */
export function dayChoices(days: readonly Day[]): Choice<string>[] {
  return days.map((day, index) => ({
    value: day.date,
    label: index === days.length - 1 ? 'Today' : dayName(day.date),
  }));
}

/** The day the form starts on: the earliest that misses its hours, else today, the report's last; none without days. */
export function defaultDay(days: readonly Day[]): Day | undefined {
  return days.find((day) => day.missing) ?? days.at(-1);
}

/** Minutes as hours and minutes: "8h", "0h 15m". */
export function minutesLabel(minutes: number): string {
  return formatHours(minutes / minutesPerHour);
}

/** The hours the form offers, in minutes: each quarter of an hour from 0h 15m to 12h. */
export const hourChoices: readonly Choice<number>[] = Array.from({ length: most / quarter }, (_, index) => {
  const minutes = (index + 1) * quarter;
  return { value: minutes, label: minutesLabel(minutes) };
});

/** A ticket as the form offers it: "WEB-9 · Fix the date picker in Safari". */
export function ticketLabel(ticket: Ticket): string {
  return `${ticket.key} · ${ticket.summary}`;
}

/** The tickets the form offers, by key, in their order. */
export function ticketChoices(tickets: readonly Ticket[]): Choice<string>[] {
  return tickets.map((ticket) => ({ value: ticket.key, label: ticketLabel(ticket) }));
}
