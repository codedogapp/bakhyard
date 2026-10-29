import { flushSync } from 'svelte';
import type { Attachment } from 'svelte/attachments';
import { SvelteMap } from 'svelte/reactivity';

import type { Ticket, Tickets } from '../jira/tickets';
import { type Added, addTodo, dismissTicket, fetchDismissals } from './api';
import type { Focus } from './focus';
import type { AddStatus, TodosState } from './state.svelte';
import { inboxTickets } from './tickets';

/** What an inbox row's buttons do: add the ticket as a todo, or dismiss it. */
export type Action = 'add' | 'dismiss';

/** A row's last add or dismiss, while it runs, once refused or once failed; one that succeeds leaves none. */
export type InboxStatus = AddStatus | { kind: 'dismissing' | 'dismissFailed' };

/**
 * The Todos page's inbox: my open tickets that no todo names and I haven't dismissed, each to add as a todo or to
 * dismiss (decisions 129, 131).
 */
export class Inbox {
  readonly #list: TodosState;
  readonly #tickets: () => Tickets;
  readonly #focus: Focus;
  /** The rows' buttons, by action and by their ticket's key; plain, since nothing on the page shows them. */
  readonly #buttons: Record<Action, Map<string, HTMLButtonElement>> = { add: new Map(), dismiss: new Map() };
  /** The keys of the tickets I've dismissed, for good; none until they have loaded. */
  dismissed = $state.raw<ReadonlySet<string>>();
  /** Whether the dismissed tickets failed to load, which leaves the inbox out. */
  failed = $state(false);
  /** The rows' adds and dismisses that run, were refused or have failed, by the ticket's key. */
  readonly statuses = new SvelteMap<string, InboxStatus>();
  /** The inbox's tickets, in the Jira section's order; none until the todos, the tickets and the dismissed have loaded. */
  readonly tickets = $derived.by(() => inboxTickets(this.#tickets(), this.#list.todos, this.dismissed));

  /** Takes the page's list, its tickets as they load, and where it gives the focus back. */
  constructor(list: TodosState, tickets: () => Tickets, focus: Focus) {
    this.#list = list;
    this.#tickets = tickets;
    this.#focus = focus;
  }

  /** Whether the dismissed tickets are still loading. */
  get loading(): boolean {
    return this.dismissed === undefined && !this.failed;
  }

  /** Loads the keys of the tickets I've dismissed. */
  async load(): Promise<void> {
    try {
      this.dismissed = new Set(await fetchDismissals());
    } catch {
      this.failed = true;
    }
  }

  /** Whether the ticket's add or dismiss is running; until it ends, another of either does nothing. */
  busy(key: string): boolean {
    const kind = this.statuses.get(key)?.kind;
    return kind === 'adding' || kind === 'dismissing';
  }

  /**
   * Adds the ticket as a todo titled with its summary and naming it, which goes on top of the list and so leaves the
   * inbox. A refused or failed add keeps the row, which says why.
   */
  async add(ticket: Ticket): Promise<void> {
    if (this.busy(ticket.key)) {
      return;
    }
    this.statuses.set(ticket.key, { kind: 'adding' });
    let added: Added;
    try {
      added = await addTodo(ticket.summary, ticket.key);
    } catch {
      this.statuses.set(ticket.key, { kind: 'failed' });
      return;
    }
    if (added.kind === 'refused') {
      this.statuses.set(ticket.key, added);
      return;
    }
    this.#leave('add', ticket.key, () => this.#list.prepend(added.todo));
  }

  /** Dismisses the ticket for good, so it leaves the inbox. A failed dismiss keeps the row, which says so. */
  async dismiss(ticket: Ticket): Promise<void> {
    if (this.busy(ticket.key)) {
      return;
    }
    this.statuses.set(ticket.key, { kind: 'dismissing' });
    try {
      await dismissTicket(ticket.key);
    } catch {
      this.statuses.set(ticket.key, { kind: 'dismissFailed' });
      return;
    }
    this.#leave('dismiss', ticket.key, () => {
      this.dismissed = new Set(this.dismissed).add(ticket.key);
    });
  }

  /** An attachment for the ticket's button for the action, which can then take the focus back. */
  button(action: Action, key: string): Attachment<HTMLButtonElement> {
    return (button) => {
      this.#buttons[action].set(key, button);
      return () => {
        this.#buttons[action].delete(key);
      };
    };
  }

  /**
   * Takes the ticket out of the inbox as `leave` does, then gives the focus to the same button of the next row, else
   * of the previous row, else to the Add field.
   */
  #leave(action: Action, key: string, leave: () => void): void {
    // Found once the request has landed, since another may have landed meanwhile.
    const neighbour = this.#neighbour(action, key);
    flushSync(() => {
      leave();
      this.statuses.delete(key);
    });
    this.#focus.recover(neighbour);
  }

  /** The action's button of the ticket after this one in the inbox, else of the one before it. */
  #neighbour(action: Action, key: string): HTMLButtonElement | undefined {
    const at = this.tickets.findIndex((ticket) => ticket.key === key);
    const neighbour = this.tickets[at + 1] ?? this.tickets[at - 1];
    return neighbour && this.#buttons[action].get(neighbour.key);
  }
}
