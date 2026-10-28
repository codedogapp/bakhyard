<script lang="ts">
import { onMount } from 'svelte';

import PageHead from '../shell/PageHead.svelte';
import { stayCurrent } from './live';
import Notifications from './Notifications.svelte';
import { type Loaded, loadState } from './nudge';
import { type Push, preparePush } from './push';
import Questions from './Questions.svelte';
import Sessions from './Sessions.svelte';

let loaded = $state.raw<Loaded>({ kind: 'loading' });
let push = $state.raw<Push>();

async function prepare() {
  push = await preparePush();
}

// A load that fails keeps what the page already shows, so a break in the connection doesn't empty it.
async function refresh() {
  const next = await loadState();
  if (next.kind === 'loaded' || loaded.kind !== 'loaded') {
    loaded = next;
  }
}

onMount(() => {
  void refresh();
  void prepare();
  return stayCurrent(() => void refresh());
});
</script>

<PageHead summary="What your agents ask, and how their work goes.">Nudge</PageHead>
<Notifications {push} changed={prepare} />
{#if loaded.kind === 'loaded'}
  <Questions questions={loaded.state.questions} answered={refresh} />
  <Sessions sessions={loaded.state.sessions} />
{:else if loaded.kind === 'failed'}
  <p class="failed">Couldn't load the questions.</p>
{/if}

<style>
.failed {
  margin: 0;
  padding: 0 var(--page-padding) 64px;
  color: var(--ink-muted);
}
</style>
