<script lang="ts">
import TicketStatus from './TicketStatus.svelte';
import type { Ticket } from './tickets';

// The key a todo names, and its ticket when that's one of my open tickets.
let { key, ticket }: { key: string; ticket: Ticket | undefined } = $props();
</script>

<span class="badge">
  <span>{key}</span>
  {#if ticket}
    <TicketStatus {ticket} />
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
</style>
