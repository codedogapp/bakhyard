<script lang="ts">
import PageHead from '../shell/PageHead.svelte';
import type { Home } from './api';
import { dateLabel, greeting, summary } from './greeting';

let { now, home }: { now: Date; home?: Home } = $props();
</script>

<!-- The punctuation sits inside the braces, so Svelte adds no `?? ''` fallback that no test can reach. -->
<PageHead eyebrow={dateLabel(now)} summary={home ? summary(home.items) : undefined}>
  {#if home}
    {`${greeting(now)}, `}<em>{home.name}</em>.
  {:else}
    {`${greeting(now)}.`}
  {/if}
</PageHead>

<style>
em {
  /* The italic capital leans into the comma; this restores a word space. */
  margin-inline-start: 0.06em;
  font-family: var(--font-serif);
  font-weight: 400;
  letter-spacing: 0;
}
</style>
