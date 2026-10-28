<script lang="ts">
import type { Home } from './api';
import Hero from './Hero.svelte';
import NeedsYou from './NeedsYou.svelte';
import ThisWeek from './ThisWeek.svelte';
import YourApps from './YourApps.svelte';

// Until the Home first loads, only the greeting and the apps show; the week, only when the Home has one.
let { now, home }: { now: Date; home?: Home } = $props();
</script>

<Hero {now} {home} />
{#if home}
  <NeedsYou {now} items={home.items} />
{/if}
<YourApps items={home?.items} />
{#if home?.week}
  <ThisWeek week={home.week} />
{/if}
