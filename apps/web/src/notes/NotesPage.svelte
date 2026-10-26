<script lang="ts">
import { onMount } from 'svelte';

import PageHead from '../shell/PageHead.svelte';
import { notesSummary, syncLabel } from './labels';
import Matches from './Matches.svelte';
import type { NotesState } from './state.svelte';
import Tree from './Tree.svelte';
import { matches, noteTree } from './tree';

// The state comes from App, which keeps it while it runs; the list loads the first time the page shows.
let { notes }: { notes: NotesState } = $props();

const tree = $derived(notes.files && noteTree(notes.files));
const filtering = $derived(notes.filter.trim() !== '');

onMount(() => {
  void notes.load();
});
</script>

<PageHead summary={tree && notesSummary(tree.count)}>Notes</PageHead>
<div class="notes">
  <div class="tools form-row">
    <input
      class="form-field"
      type="search"
      aria-label="Filter notes"
      placeholder="Filter notes"
      bind:value={notes.filter}
    >
    <!-- aria-disabled, not disabled, so focus stays on the button while it waits. -->
    <button class="form-button" type="button" aria-disabled={notes.busy} onclick={() => notes.sync()}>Sync</button>
    <p class="form-status" role="status">{syncLabel(notes.status)}</p>
  </div>
  {#if tree && (filtering || tree.count > 0)}
    <div class="card">
      {#if filtering}
        <Matches notes={matches(tree, notes.filter)} filter={notes.filter} />
      {:else}
        <Tree folder={tree} open={notes.open} />
      {/if}
    </div>
  {:else if notes.status.kind === 'load-failed'}
    <p class="failed">Couldn't load the notes.</p>
  {/if}
</div>

<style>
.notes {
  padding: 0 var(--page-padding) 64px;
}

.tools {
  margin-bottom: 30px;
}

.card {
  border: 1px solid var(--rule);
  background: var(--surface);
  box-shadow: var(--offset) var(--offset) 0 var(--shadow);
}

.failed {
  margin: 0;
  color: var(--ink-muted);
}

@media (max-width: 700px) {
  .notes {
    padding-bottom: 48px;
  }

  .tools {
    margin-bottom: 24px;
  }
}
</style>
