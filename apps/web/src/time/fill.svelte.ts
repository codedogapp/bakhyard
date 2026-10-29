import type { Attachment } from 'svelte/attachments';
import { on } from 'svelte/events';

import { loadTickets, type Ticket, type Tickets } from '../jira/tickets';
import { focusLost } from '../shell/focus';
import { defaultDay, workday } from './choices';
import { type Day, loadReport, type TimeReport } from './report';
import { addWorklog, type Refused } from './worklog';

/** A worklog to add: the minutes on the ticket, from 09:00 on the date. */
export interface Entry {
  date: string;
  ticket: Ticket;
  minutes: number;
}

/** The form's last log: running, or how it ended. */
export type FillStatus = { kind: 'idle' | 'logging' | 'logged' | 'failed' } | Refused;

/**
 * The Time page's form, which fills a day: it adds a worklog of the hours on one of my open tickets, once its dialog has
 * asked (decision 194).
 */
export class Fill {
  /** My open tickets, which the form offers. */
  tickets = $state.raw<Tickets>({ kind: 'loading' });
  /** The day picked, as YYYY-MM-DD; none until one is, and the form shows the default. */
  picked = $state<string>();
  /** The picked ticket's key; empty until one is. */
  ticket = $state('');
  minutes = $state(workday);
  status = $state.raw<FillStatus>({ kind: 'idle' });
  /** What the dialog asks to log, as the form had it when Log opened the dialog. */
  entry = $state.raw<Entry>();
  /** Puts a report on the page. */
  readonly #show: (report: TimeReport) => void;
  /** The page's dialog, which asks before a log. */
  #dialog: HTMLDialogElement | undefined;
  /** The form's Log button, which opens the dialog. */
  #logButton: HTMLButtonElement | undefined;

  constructor(show: (report: TimeReport) => void) {
    this.#show = show;
  }

  /** My open tickets once they've loaded, in Jira's order; none before. */
  get open(): readonly Ticket[] {
    return this.tickets.kind === 'loaded' ? (this.tickets.snapshot.tickets ?? []) : [];
  }

  /** Whether a log runs; until it ends, Log opens nothing. */
  get busy(): boolean {
    return this.status.kind === 'logging';
  }

  /** Loads my open tickets. */
  async load(): Promise<void> {
    this.tickets = await loadTickets();
  }

  /** The day the form shows: the picked one while the report has it, else the default. */
  day(days: readonly Day[]): string | undefined {
    return (days.find(({ date }) => date === this.picked) ?? defaultDay(days))?.date;
  }

  /** Asks in the page's dialog whether to log the hours on the ticket on the day, as the form has them. */
  confirm(days: readonly Day[]): void {
    const date = this.day(days);
    const ticket = this.open.find(({ key }) => key === this.ticket);
    if (this.busy || date === undefined || ticket === undefined) {
      return;
    }
    this.entry = { date, ticket, minutes: this.minutes };
    this.#dialog?.showModal();
  }

  /** Closes the dialog without logging. */
  cancel(): void {
    this.#dialog?.close();
  }

  /**
   * Closes the dialog and adds the worklog, then loads the report again and puts it on the page if it loaded. The form
   * keeps its values, and the day it logged rather than move on to the next missing one.
   */
  async log(): Promise<void> {
    const entry = this.entry;
    this.#dialog?.close();
    if (this.busy || entry === undefined) {
      return;
    }
    this.picked = entry.date;
    this.status = { kind: 'logging' };
    try {
      const added = await addWorklog(entry.ticket.key, entry.date, entry.minutes);
      if (added.kind === 'refused') {
        this.status = added;
        return;
      }
    } catch {
      this.status = { kind: 'failed' };
      return;
    }
    const report = await loadReport();
    if (report.kind === 'loaded') {
      this.#show(report);
    }
    this.status = { kind: 'logged' };
  }

  /**
   * An attachment for the page's dialog. As it closes, the focus goes back to the form's Log button where the browser
   * didn't put it back: WebKit doesn't after a click, which focuses no button.
   */
  readonly dialog: Attachment<HTMLDialogElement> = (dialog) => {
    this.#dialog = dialog;
    return on(dialog, 'close', () => {
      if (focusLost()) {
        this.#logButton?.focus();
      }
    });
  };

  /** An attachment for the form's Log button. */
  readonly logButton: Attachment<HTMLButtonElement> = (button) => {
    this.#logButton = button;
  };
}
