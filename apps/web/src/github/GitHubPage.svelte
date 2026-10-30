<script lang="ts">
import { onMount } from 'svelte';

import PageHead from '../shell/PageHead.svelte';
import PolledSection from '../shell/PolledSection.svelte';
import { type Item, type Items, type List, loadItems, type Snapshot } from './github';

const loading: Items = { kind: 'loading' };
let lists = $state.raw<Record<List, Items>>({ mine: loading, reviews: loading, issues: loading });

// The lists load at once, so none waits for another, as on the PRs page (decision 172).
onMount(() => {
  for (const list of ['mine', 'reviews', 'issues'] as const) {
    void loadItems(list).then((loaded) => {
      lists = { ...lists, [list]: loaded };
    });
  }
});

const itemsOf = (snapshot: Snapshot) => snapshot.items;
</script>

{#snippet itemRow(item: Item)}
  <!-- An item without a link to GitHub gets a row that isn't a link. -->
  <svelte:element this={item.url ? 'a' : 'div'} class="list-row" href={item.url}>
    <span class="list-label">#{item.id}</span>
    <span class="list-text">
      <span class="list-title">{item.title}</span>
      <span class="list-detail">{item.repo}</span>
    </span>
    <span class="list-end">
      {#if item.draft}
        <span class="chip">Draft</span>
      {/if}
    </span>
  </svelte:element>
{/snippet}

<PageHead>GitHub</PageHead>
<!-- Without GitHub, the github service answers 404 for every list. -->
{#if Object.values(lists).every((items) => items.kind === 'off')}
  <p class="page-status">GitHub isn't set up.</p>
{:else}
  <PolledSection
    polled={lists.mine}
    list={itemsOf}
    title="Your"
    accent="PRs"
    noun="PR"
    empty="No open pull requests."
    failed="Couldn't load the pull requests."
    row={itemRow}
  />
  <PolledSection
    polled={lists.reviews}
    list={itemsOf}
    title="Review"
    accent="requests"
    noun="PR"
    empty="No pull requests to review."
    failed="Couldn't load the review requests."
    row={itemRow}
  />
  <PolledSection
    polled={lists.issues}
    list={itemsOf}
    title="Assigned"
    accent="issues"
    noun="issue"
    empty="No open issues."
    failed="Couldn't load the issues."
    row={itemRow}
  />
{/if}
