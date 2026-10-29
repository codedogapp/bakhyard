import type { Ticket, Tickets } from '../jira/tickets';
import type { Todo } from './api';

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
