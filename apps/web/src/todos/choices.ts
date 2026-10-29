import type { Ticket } from '../jira/tickets';

/** One of the ticket picker's options: the key it links, empty for none, and its text. */
export interface Choice {
  key: string;
  label: string;
}

/**
 * The tickets the editor offers a todo: none, then the todo's own if it isn't one of my open tickets, by its key alone
 * as the row's badge shows it, then my open tickets in Jira's order, each with its summary.
 */
export function ticketChoices(open: ReadonlyMap<string, Ticket>, linked: string | undefined): Choice[] {
  const choices: Choice[] = [{ key: '', label: 'No ticket' }];
  if (linked && !open.has(linked)) {
    choices.push({ key: linked, label: linked });
  }
  for (const { key, summary } of open.values()) {
    choices.push({ key, label: `${key} · ${summary}` });
  }
  return choices;
}
