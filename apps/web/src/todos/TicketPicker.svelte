<script lang="ts">
import { ticketChoices } from './choices';
import type { Editor } from './editor.svelte';
import type { Ticket } from './tickets';

// `open` holds my open tickets by key, in Jira's order.
let { editor, open }: { editor: Editor; open: ReadonlyMap<string, Ticket> } = $props();

const choices = $derived(ticketChoices(open, editor.todo?.ticket));
</script>

<!-- Only when there's a choice: my open tickets to link, or the todo's own ticket to unlink. -->
{#if choices.length > 1}
  <span class="ticket">
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
  position: relative;
  display: flex;
  flex: 1 1 14em;
  min-width: 0;
}

/* Room on the right for the chevron, 14 px in from the edge as the text is from the left. */
select {
  padding-right: 44px;
  text-overflow: ellipsis;
  cursor: pointer;
}

svg {
  position: absolute;
  top: 50%;
  right: 14px;
  translate: 0 -50%;
  fill: none;
  stroke: currentColor;
  stroke-linecap: square;
  stroke-width: 1.5;
  pointer-events: none;
}

/* Muted while a save, a delete or a move runs, as the buttons and the priorities are. */
.ticket:has(:disabled) {
  color: var(--ink-muted);
}

/* Chrome also fades a disabled select; the muted ink alone says it waits, as for the buttons. */
select:disabled {
  opacity: 1;
  cursor: default;
}
</style>
