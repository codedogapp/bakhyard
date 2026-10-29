<script lang="ts">
import type { Ticket } from '../jira/tickets';

let { ticket }: { ticket: Ticket } = $props();
</script>

<span class={['chip', { 'chip-filled': ticket.category === 'in-progress' }]}>{ticket.status}</span>

<style>
/* A long status wraps in its chip, at most 160 px wide, rather than squeeze what's beside it or widen the row. */
span {
  max-width: 160px;
  overflow-wrap: anywhere;
  white-space: normal;
}
</style>
