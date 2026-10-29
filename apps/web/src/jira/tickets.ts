import { loadSnapshot, type Polled, type Polls } from '../shell/snapshot';

/** A ticket's status category, as the todos service names Jira's; "unknown" stands for any other. */
type Category = 'to-do' | 'in-progress' | 'done' | 'unknown';

/** A Jira ticket, as the todos service lists it: one of my open tickets, or a done one that a todo names. */
export interface Ticket {
  key: string;
  summary: string;
  /** The status's name, as the ticket's workflow calls it. */
  status: string;
  category: Category;
  /** The ticket's page in Jira. */
  url: string;
}

/** The body of GET /api/todos/tickets: what the todos service last found in Jira (decisions 122, 133). */
export interface Snapshot extends Polls {
  /** My open tickets, from the last poll that succeeded, in Jira's order; none before the first. */
  tickets?: Ticket[];
  /** The tickets that the todos left to do name and that are done, from that poll; none when there are none. */
  done?: Ticket[];
}

/** The tickets as a page loads them: loading, off when Jira isn't set up, failed to load, or loaded. */
export type Tickets = Polled<Snapshot>;

/** Loads the tickets; never fails, since a failed load is one of the outcomes. */
export function loadTickets(): Promise<Tickets> {
  return loadSnapshot('/api/todos/tickets');
}
