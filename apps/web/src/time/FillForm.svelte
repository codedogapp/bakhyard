<script lang="ts">
import { type Choice, dayChoices, hourChoices, ticketChoices } from './choices';
import type { Fill } from './fill.svelte';
import { fillLabel, ticketsNote } from './labels';
import type { Day } from './report';

// `days` holds the report's days, which the form offers.
let { fill, days }: { fill: Fill; days: readonly Day[] } = $props();

function submit(event: SubmitEvent): void {
  event.preventDefault();
  fill.confirm(days);
}
</script>

{#snippet options(choices: readonly Choice<string | number>[])}
  {#each choices as { value, label } (value)}
    <option {value}>{label}</option>
  {/each}
{/snippet}

{#snippet chevron()}
  <!-- The arrow's head, pointing down, in place of the browser's own. -->
  <svg aria-hidden="true" viewBox="0 0 16 16" width="16" height="16"><path d="M4 6l4 4 4-4" /></svg>
{/snippet}

<!-- Log only asks, in the page's dialog; the dialog's Log adds the worklog (decision 194). -->
{#if fill.open.length > 0}
  <form class="fill" onsubmit={submit}>
    <div class="form-row">
      <span class="form-select day">
        <select class="form-field" aria-label="Day" bind:value={() => fill.day(days), (date) => (fill.picked = date)}>
          {@render options(dayChoices(days))}
        </select>
        {@render chevron()}
      </span>
      <!-- required, so Log with no ticket picked has the browser ask for one. -->
      <span class="form-select ticket">
        <select class="form-field" aria-label="Ticket" required bind:value={fill.ticket}>
          <option value="">Pick a ticket</option>
          {@render options(ticketChoices(fill.open))}
        </select>
        {@render chevron()}
      </span>
      <!-- The hours and Log wrap together, so Log never sits alone on a line. -->
      <span class="end">
        <span class="form-select hours">
          <select class="form-field" aria-label="Hours" bind:value={fill.minutes}>
            {@render options(hourChoices)}
          </select>
          {@render chevron()}
        </span>
        <!-- aria-disabled, not disabled, so the focus stays on the button while a log runs. -->
        <button class="form-button" type="submit" aria-disabled={fill.busy} {@attach fill.logButton}>Log</button>
      </span>
    </div>
    <p class="form-status" role="status">{fillLabel(fill.status)}</p>
  </form>
{:else}
  {@const note = ticketsNote(fill.tickets)}
  {#if note}
    <p class="note">{note}</p>
  {/if}
{/if}

<style>
/* Over the sections, the form leaves the space that the Todos page's form leaves over its first section. */
.fill,
.note {
  margin: 0 0 30px;
  padding: 0 var(--page-padding);
}

.form-row {
  row-gap: 12px;
}

/* The day as wide as its longest name, in any font; the ticket's summary takes most of the room to spare. */
.day {
  flex: 1 1 auto;
}

.ticket {
  flex: 3 1 14em;
}

.end {
  display: flex;
  gap: 12px;
}

.hours {
  flex: none;
  width: 8em;
}

.note {
  color: var(--ink-muted);
  overflow-wrap: anywhere;
}

@media (max-width: 700px) {
  .fill,
  .note {
    margin-bottom: 24px;
  }
}
</style>
