<script lang="ts">
import Section from '../shell/Section.svelte';
import { searchedLabel, ticketsNote } from './labels';
import type { Tickets } from './tickets';

let { tickets }: { tickets: Tickets } = $props();
</script>

{#if tickets.kind === 'loaded'}
  {@const { tickets: list, searched, failure } = tickets.snapshot}
  <Section title="Jira" accent="tickets" note={list && ticketsNote(list.length)}>
    {#if failure}
      <p class="muted">
        The last search failed: {failure}.
        {#if list && searched}
          These tickets are from <time datetime={searched}>{searchedLabel(searched)}</time>.
        {/if}
      </p>
    {/if}
    {#if list}
      <div class="card">
        {#if list.length > 0}
          <ul class="list">
            {#each list as ticket}
              <li>
                <a class="list-row" href={ticket.url}>
                  <span class="list-label">{ticket.key}</span>
                  <span class="list-text"><span class="list-title">{ticket.summary}</span></span>
                  <span class="list-end">
                    <span class={['chip', 'status', { 'chip-filled': ticket.category === 'in-progress' }]}>
                      {ticket.status}
                    </span>
                  </span>
                </a>
              </li>
            {/each}
          </ul>
        {:else}
          <p class="empty">No open tickets.</p>
        {/if}
      </div>
    {/if}
  </Section>
{:else if tickets.kind === 'failed'}
  <Section title="Jira" accent="tickets">
    <p class="muted">Couldn't load the tickets.</p>
  </Section>
{/if}

<style>
.muted {
  margin: 0;
  color: var(--ink-muted);
  overflow-wrap: anywhere;
}

.card {
  border: 1px solid var(--rule);
  background: var(--surface);
  box-shadow: var(--offset) var(--offset) 0 var(--shadow);
}

.muted + .card {
  margin-top: 20px;
}

/* A long status wraps in its chip, rather than squeezing the summary or widening the row. */
.status {
  max-width: 160px;
  overflow-wrap: anywhere;
  white-space: normal;
}

.empty {
  margin: 0;
  padding: 20px;
  color: var(--ink-muted);
}
</style>
