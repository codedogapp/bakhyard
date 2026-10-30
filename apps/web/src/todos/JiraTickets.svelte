<script lang="ts">
import type { Tickets } from '../jira/tickets';
import PolledSection from '../shell/PolledSection.svelte';
import TicketStatus from './TicketStatus.svelte';

let { tickets }: { tickets: Tickets } = $props();
</script>

<PolledSection
  polled={tickets}
  list={(snapshot) => snapshot.tickets}
  title="Jira"
  accent="tickets"
  noun="ticket"
  empty="No open tickets."
  failed="Couldn't load the tickets."
>
  {#snippet row(ticket)}
    <a class="list-row" href={ticket.url}>
      <span class="list-label">{ticket.key}</span>
      <span class="list-text"><span class="list-title">{ticket.summary}</span></span>
      <span class="list-end"><TicketStatus {ticket} /></span>
    </a>
  {/snippet}
</PolledSection>
