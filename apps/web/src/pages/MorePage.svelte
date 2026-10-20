<script lang="ts">
import AppCards from '../home/AppCards.svelte';
import type { Item } from '../home/api';
import { moreApps } from '../shell/nav';
import PageHead from '../shell/PageHead.svelte';

// The counts come with the Home's first load, as on the Home's cards.
let { items }: { items?: readonly Item[] } = $props();
</script>

<PageHead summary="The apps without a tab of their own.">More</PageHead>
<div class="apps">
  <AppCards apps={moreApps} {items} headingLevel={2} />
</div>

<style>
.apps {
  padding: 0 var(--page-padding) 64px;
}

@media (max-width: 700px) {
  .apps {
    padding-bottom: 48px;
  }
}
</style>
