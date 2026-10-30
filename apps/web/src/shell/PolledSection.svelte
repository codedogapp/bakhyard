<script lang="ts" generics="S extends import('./snapshot').Polls, T">
import type { Snippet } from 'svelte';

import { countLabel } from './count';
import Section from './Section.svelte';
import { type Polled, searchedLabel } from './snapshot';

// A poller's list in a section under a page's head, as the PRs page's lists and the Todos page's Jira tickets are: a row
// for each item, in the list's order, and texts that say when it's empty, why the last search failed and when it
// couldn't load; nothing while it loads or when its source isn't set up (decision 205). Biome doesn't read the generics
// attribute (biomejs/biome#11215), so it names Polls with import() rather than an import that Biome takes for unused.
let {
  polled,
  list,
  title,
  accent,
  noun,
  empty,
  failed,
  row,
}: {
  polled: Polled<S>;
  /** The snapshot's list, such as its PRs; none before a search has succeeded. */
  list: (snapshot: S) => readonly T[] | undefined;
  title: string;
  accent: string;
  /** What the list holds, such as PR: the head counts "3 PRs", and the failure line says "These PRs are from". */
  noun: string;
  empty: string;
  failed: string;
  row: Snippet<[T]>;
} = $props();
</script>

{#if polled.kind === 'loaded'}
  {@const { searched, failure } = polled.snapshot}
  {@const items = list(polled.snapshot)}
  <Section {title} {accent} note={items && countLabel(items.length, noun)}>
    {#if failure}
      <p class="muted">
        The last search failed: {failure}.
        {#if items && searched}
          These {noun}s are from <time datetime={searched}>{searchedLabel(searched)}</time>.
        {/if}
      </p>
    {/if}
    {#if items}
      <div class="card">
        {#if items.length > 0}
          <ul class="list">
            {#each items as item}
              <li>{@render row(item)}</li>
            {/each}
          </ul>
        {:else}
          <p class="empty">{empty}</p>
        {/if}
      </div>
    {/if}
  </Section>
{:else if polled.kind === 'failed'}
  <Section {title} {accent}>
    <p class="muted">{failed}</p>
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

.empty {
  margin: 0;
  padding: 20px;
  color: var(--ink-muted);
}
</style>
