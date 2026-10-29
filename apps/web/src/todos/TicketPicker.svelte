<script lang="ts">
import type { Ticket } from '../jira/tickets';
import { ticketChoices } from './choices';
import type { Editor } from './editor.svelte';

// `open` holds my open tickets by key, in Jira's order.
let { editor, open }: { editor: Editor; open: ReadonlyMap<string, Ticket> } = $props();

const choices = $derived(ticketChoices(open, editor.todo?.ticket));
</script>

<!-- Only when there's a choice: my open tickets to link, or the todo's own ticket to unlink. -->
{#if choices.length > 1}
  <span class="form-select ticket">
    <!-- disabled, not aria-disabled, while a save, a delete or a move runs: a select has no readonly. -->
    <select class="form-field" aria-label="Ticket" disabled={editor.busy} bind:value={editor.draft.ticket}>
      {#each choices as { key, label } (key)}
        <option value={key}>{label}</option>
      {/each}
    </select>
    <!-- The arrow's head, pointing down, in place of the browser's own. -->
    <svg aria-hidden="true" viewBox="0 0 16 16" width="16" height="16"><path d="M4 6l4 4 4-4" /></svg>
  </span>
{/if}

<style>
/* Wide enough for a key and the start of its summary, and wider where the line has room. */
.ticket {
  flex: 1 1 14em;
}
</style>
