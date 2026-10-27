import type { Todo } from './api';

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
export interface Snapshot {
  /** My open tickets, from the last poll that succeeded, in Jira's order; none before the first. */
  tickets?: Ticket[];
  /** The tickets that the todos left to do name and that are done, from that poll; none when there are none. */
  done?: Ticket[];
  /** When that poll began, in UTC. */
  searched?: string;
  /** Why the last poll failed, if it did. */
  failure?: string;
}

/** The Todos page's tickets: loading, off when Jira isn't set up, failed to load, or loaded. */
export type Tickets = { kind: 'loading' | 'off' | 'failed' } | { kind: 'loaded'; snapshot: Snapshot };

function byKey(list: readonly Ticket[] = []): ReadonlyMap<string, Ticket> {
  return new Map(list.map((ticket) => [ticket.key, ticket]));
}

/** My open tickets by key, from the last poll that succeeded; none while loading, after a failed load, or without Jira. */
export function openTickets(tickets: Tickets): ReadonlyMap<string, Ticket> {
  return byKey(tickets.kind === 'loaded' ? tickets.snapshot.tickets : undefined);
}

/** The done tickets that the todos left to do name, by key, from the last poll that succeeded (decision 134). */
export function doneTickets(tickets: Tickets): ReadonlyMap<string, Ticket> {
  return byKey(tickets.kind === 'loaded' ? tickets.snapshot.done : undefined);
}

/**
 * The inbox's tickets: my open tickets, from the last poll that succeeded, that no todo names, a done one included,
 * and that I haven't dismissed, in the Jira section's order (decisions 129, 131); none until the todos and the
 * dismissed tickets have loaded.
 */
export function inboxTickets(
  tickets: Tickets,
  todos: readonly Todo[] | undefined,
  dismissed: ReadonlySet<string> | undefined
): Ticket[] {
  if (todos === undefined || dismissed === undefined || tickets.kind !== 'loaded') {
    return [];
  }
  const named = new Set(todos.map((todo) => todo.ticket));
  return (tickets.snapshot.tickets ?? []).filter((ticket) => !named.has(ticket.key) && !dismissed.has(ticket.key));
}

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
