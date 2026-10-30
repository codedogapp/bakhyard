<script lang="ts">
import { onMount } from 'svelte';

import PageHead from '../shell/PageHead.svelte';
import { stayCurrent } from './live';
import { type Loaded, loadState } from './nudge';
import Questions from './Questions.svelte';
import Sessions from './Sessions.svelte';

let loaded = $state.raw<Loaded>({ kind: 'loading' });

// A load that fails keeps what the page already shows, so a break in the connection doesn't empty it.
async function refresh() {
  const next = await loadState();
  if (next.kind === 'loaded' || loaded.kind !== 'loaded') {
    loaded = next;
  }
}

onMount(() => {
  void refresh();
  return stayCurrent(() => void refresh());
});
</script>

<PageHead summary="What your agents ask, and how their work goes.">Nudge</PageHead>
{#if loaded.kind === 'loaded'}
  <Questions questions={loaded.state.questions} answered={refresh} />
  <Sessions sessions={loaded.state.sessions} />
{:else if loaded.kind === 'failed'}
  <p class="page-status">Couldn't load the questions.</p>
{/if}
