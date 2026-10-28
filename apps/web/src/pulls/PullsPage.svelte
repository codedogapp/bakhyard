<script lang="ts">
import { onMount } from 'svelte';

import PageHead from '../shell/PageHead.svelte';
import { loadPulls, type Pulls } from './pulls';
import YourPulls from './YourPulls.svelte';

let pulls = $state.raw<Pulls>({ kind: 'loading' });

onMount(() => {
  void loadPulls().then((loaded) => {
    pulls = loaded;
  });
});
</script>

<PageHead>Pull requests</PageHead>
{#if pulls.kind === 'off'}
  <p class="off">Bitbucket isn't set up.</p>
{:else}
  <YourPulls {pulls} />
{/if}

<style>
.off {
  margin: 0;
  padding: 0 var(--page-padding) 64px;
  color: var(--ink-muted);
}

@media (max-width: 700px) {
  .off {
    padding-bottom: 48px;
  }
}
</style>
