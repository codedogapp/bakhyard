<script lang="ts">
import TicketStatus from './TicketStatus.svelte';
import type { Ticket } from './tickets';

// The key a todo names, its ticket when that's one of my open tickets or a done one, and whether the todo is ticked.
let { key, ticket, ticked }: { key: string; ticket: Ticket | undefined; ticked: boolean } = $props();
</script>

<span class="badge">
  <span>{key}</span>
  {#if ticket}
    <TicketStatus {ticket} />
    {#if ticket.category === 'done' && !ticked}
      <!-- The offer to tick the todo (decision 134): it's in the box's label, so a tap on it ticks. -->
      <span class="offer">Done in Jira</span>
    {/if}
  {/if}
</span>

<style>
/* The list rows' mono label (keys are in capitals), on lines as tall as a chip, so a late status moves nothing. */
.badge {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 10px;
  align-items: center;
  color: var(--ink-muted);
  font: 400 11px / 20px var(--font-mono);
  letter-spacing: 0.045em;
}

/* In the ink rather than muted, so it stands out from the key, and in capitals, as mono text always is. */
.offer {
  color: var(--ink);
  text-transform: uppercase;
}
</style>
