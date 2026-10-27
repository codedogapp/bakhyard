import { flushSync } from 'svelte';
import type { Attachment } from 'svelte/attachments';
import { SvelteMap } from 'svelte/reactivity';

import { type Added, addTodo } from './api';
import type { Focus } from './focus';
import type { AddStatus, TodosState } from './state.svelte';
import { inboxTickets, type Ticket, type Tickets } from './tickets';

/** The Todos page's inbox: my open tickets that no todo names, each to add as a todo (decision 129). */
export class Inbox {
  readonly #list: TodosState;
  readonly #tickets: () => Tickets;
  readonly #focus: Focus;
  /** The rows' Add buttons, by their ticket's key; plain, since nothing on the page shows it. */
  readonly #buttons = new Map<string, HTMLButtonElement>();
  /** The adds that run, were refused or have failed, by the ticket's key; an add that succeeds leaves none. */
  readonly adds = new SvelteMap<string, AddStatus>();
  /** The inbox's tickets, in the Jira section's order; none until both the todos and the tickets have loaded. */
  readonly tickets = $derived.by(() => inboxTickets(this.#tickets(), this.#list.todos));

  /** Takes the page's list, its tickets as they load, and where it gives the focus back. */
  constructor(list: TodosState, tickets: () => Tickets, focus: Focus) {
    this.#list = list;
    this.#tickets = tickets;
    this.#focus = focus;
  }

  /** Whether the ticket's add is running; until it ends, another add of it does nothing. */
  adding(key: string): boolean {
    return this.adds.get(key)?.kind === 'adding';
  }

  /**
   * Adds the ticket as a todo titled with its summary and naming it, which goes on top of the list and so leaves the
   * inbox, then gives the focus to the next row's Add button, else the previous row's, else the Add field. A refused
   * or failed add keeps the row, which says why.
   */
  async add(ticket: Ticket): Promise<void> {
    if (this.adding(ticket.key)) {
      return;
    }
    this.adds.set(ticket.key, { kind: 'adding' });
    let added: Added;
    try {
      added = await addTodo(ticket.summary, ticket.key);
    } catch {
      this.adds.set(ticket.key, { kind: 'failed' });
      return;
    }
    if (added.kind === 'refused') {
      this.adds.set(ticket.key, added);
      return;
    }
    // Found once the add has landed, since another may have landed meanwhile.
    const neighbour = this.#neighbour(ticket.key);
    flushSync(() => {
      this.#list.prepend(added.todo);
      this.adds.delete(ticket.key);
    });
    this.#focus.recover(neighbour);
  }

  /** An attachment for the ticket's Add button, which can then take the focus back. */
  button(key: string): Attachment<HTMLButtonElement> {
    return (button) => {
      this.#buttons.set(key, button);
      return () => {
        this.#buttons.delete(key);
      };
    };
  }

  /** The Add button of the ticket after this one in the inbox, else of the one before it. */
  #neighbour(key: string): HTMLButtonElement | undefined {
    const at = this.tickets.findIndex((ticket) => ticket.key === key);
    const neighbour = this.tickets[at + 1] ?? this.tickets[at - 1];
    return neighbour && this.#buttons.get(neighbour.key);
  }
}
