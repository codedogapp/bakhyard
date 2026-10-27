<script lang="ts">
import Section from '../shell/Section.svelte';
import type { Inbox } from './inbox.svelte';
import { addLabel, ticketsNote } from './labels';
import TicketStatus from './TicketStatus.svelte';

let { inbox }: { inbox: Inbox } = $props();
</script>

<!-- No section without a ticket: there's nothing to decide. -->
{#if inbox.tickets.length > 0}
  <Section title="Jira" accent="inbox" note={ticketsNote(inbox.tickets.length)}>
    <ul class="list card">
      {#each inbox.tickets as ticket (ticket.key)}
        <li class="list-row">
          <span class="list-label">{ticket.key}</span>
          <div class="list-text">
            <span class="list-title">{ticket.summary}</span>
            <!-- Never hidden: a live region has to be there before its first change for screen readers to announce it. -->
            <p class="status" role="status">{addLabel(inbox.adds.get(ticket.key))}</p>
          </div>
          <span class="list-end">
            <TicketStatus {ticket} />
            <!-- aria-disabled, not disabled, so focus stays on the button while its add runs. -->
            <button
              class="form-button"
              type="button"
              aria-label="Add {ticket.key}"
              aria-disabled={inbox.adding(ticket.key)}
              onclick={() => inbox.add(ticket)}
              {@attach inbox.button(ticket.key)}
            >
              Add
            </button>
          </span>
        </li>
      {/each}
    </ul>
  </Section>
{/if}

<style>
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
</style>
