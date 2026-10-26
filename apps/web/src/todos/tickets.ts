/** A ticket's status category, as the todos service names Jira's; "unknown" stands for any other. */
type Category = 'to-do' | 'in-progress' | 'done' | 'unknown';

/** One of my open Jira tickets, as the todos service lists it. */
export interface Ticket {
  key: string;
  summary: string;
  /** The status's name, as the ticket's workflow calls it. */
  status: string;
  category: Category;
  /** The ticket's page in Jira. */
  url: string;
}

/** The body of GET /api/todos/tickets: what the todos service last found in Jira (decision 122). */
export interface Snapshot {
  /** The tickets of the last search that succeeded, in Jira's order; none before the first. */
  tickets?: Ticket[];
  /** When that search began, in UTC. */
  searched?: string;
  /** Why the last search failed, if it did. */
  failure?: string;
}

/** The Todos page's tickets: loading, off when Jira isn't set up, failed to load, or loaded. */
export type Tickets = { kind: 'loading' | 'off' | 'failed' } | { kind: 'loaded'; snapshot: Snapshot };

/** Loads the tickets; never fails, since a failed load is one of the outcomes. */
export async function loadTickets(): Promise<Tickets> {
  try {
    const response = await fetch('/api/todos/tickets');
    if (response.status === 404) {
      return { kind: 'off' };
    }
    if (response.status !== 200) {
      return { kind: 'failed' };
    }
    const snapshot: Snapshot = await response.json();
    // Only a request that waited while the service stopped gets neither.
    if (snapshot.tickets === undefined && snapshot.failure === undefined) {
      return { kind: 'failed' };
    }
    return { kind: 'loaded', snapshot };
  } catch {
    return { kind: 'failed' };
  }
}
