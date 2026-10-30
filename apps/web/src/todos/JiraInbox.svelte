<script lang="ts">
import { countLabel } from '../shell/count';
import Section from '../shell/Section.svelte';
import type { Inbox } from './inbox.svelte';
import { inboxLabel } from './labels';
import TicketStatus from './TicketStatus.svelte';

let { inbox }: { inbox: Inbox } = $props();
</script>

<!-- No section without a ticket: there's nothing to decide. -->
{#if inbox.tickets.length > 0}
  <Section title="Jira" accent="inbox" note={countLabel(inbox.tickets.length, 'ticket')}>
    <ul class="list card">
      {#each inbox.tickets as ticket (ticket.key)}
        <li class="list-row">
          <span class="list-label">{ticket.key}</span>
          <div class="list-text">
            <span class="list-title">{ticket.summary}</span>
            <!-- Never hidden: a live region has to be there before its first change for screen readers to announce it. -->
            <p class="status" role="status">{inboxLabel(inbox.statuses.get(ticket.key))}</p>
          </div>
          <span class="list-end"><TicketStatus {ticket} /></span>
          <!-- aria-disabled, not disabled, so focus stays on the button while its add or dismiss runs. -->
          <span class="actions">
            <button
              class="form-button"
              type="button"
              aria-label={`Add ${ticket.key}`}
              aria-disabled={inbox.busy(ticket.key)}
              onclick={() => inbox.add(ticket)}
              {@attach inbox.button('add', ticket.key)}
            >
              Add
            </button>
            <button
              class="form-button"
              type="button"
              aria-label={`Dismiss ${ticket.key}`}
              aria-disabled={inbox.busy(ticket.key)}
              onclick={() => inbox.dismiss(ticket)}
              {@attach inbox.button('dismiss', ticket.key)}
            >
              Dismiss
            </button>
          </span>
        </li>
      {/each}
    </ul>
  </Section>
{/if}

<style>
/* The buttons after the chip; on a phone, on a line of their own under the summary. */
.list-row {
  grid-template-areas: "label text end actions";
  grid-template-columns: 120px 1fr auto auto;
}

.actions {
  display: flex;
  gap: 10px;
  grid-area: actions;
}

.card {
  border: 1px solid var(--rule);
  background: var(--surface);
  box-shadow: var(--offset) var(--offset) 0 var(--shadow);
}

/* The status line under the summary, once it has something to say, in the mono status style. */
.status {
  margin: 0;
  color: var(--ink-muted);
  font: 400 11px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
}

.status:not(:empty) {
  margin-top: 4px;
}

@media (max-width: 700px) {
  .list-row {
    grid-template-areas:
      "label end"
      "text text"
      "actions actions";
    grid-template-columns: 1fr auto;
  }
}
</style>
