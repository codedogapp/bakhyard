import type { Tickets } from '../jira/tickets';
import { minutesLabel, ticketLabel } from './choices';
import { dayName } from './days';
import type { Entry, FillStatus } from './fill.svelte';

/** What the dialog asks to log: "8h on WEB-9 · Fix the date picker in Safari, from 09:00 on Friday 16 Oct." */
export function entryText({ date, ticket, minutes }: Entry): string {
  return `${minutesLabel(minutes)} on ${ticketLabel(ticket)}, from 09:00 on ${dayName(date)}.`;
}

/** What the last log did: "Logging…", "Logged", the service's reason or "Log failed"; blank before the first. */
export function fillLabel(status: FillStatus): string {
  switch (status.kind) {
    case 'logging':
      return 'Logging…';
    case 'logged':
      return 'Logged';
    case 'refused':
      return status.reason;
    case 'failed':
      return 'Log failed';
    default:
      return '';
  }
}

/**
 * What the Time page says in the form's place: that my open tickets couldn't load, as when the todos service has no
 * Jira or hasn't searched it yet, or that none is open; nothing while they load, or once there are tickets to offer.
 */
export function ticketsNote(tickets: Tickets): string | undefined {
  if (tickets.kind === 'loading') {
    return undefined;
  }
  if (tickets.kind !== 'loaded' || tickets.snapshot.tickets === undefined) {
    return "Couldn't load your Jira tickets.";
  }
  return tickets.snapshot.tickets.length === 0 ? 'No open Jira tickets to log hours on.' : undefined;
}
