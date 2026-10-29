<script lang="ts">
import type { Home } from './api';
import Hero from './Hero.svelte';
import NeedsYou from './NeedsYou.svelte';
import ThisWeek from './ThisWeek.svelte';
import YourApps from './YourApps.svelte';

// Until the Home first loads, only the greeting and the apps show; the week, when the Home has one, and its
// head with "Hours unavailable" when home couldn't get it from the time service (decision 187).
let { now, home }: { now: Date; home?: Home } = $props();
</script>

<Hero {now} {home} />
{#if home}
  <NeedsYou {now} items={home.items} />
{/if}
<YourApps items={home?.items} />
{#if home?.week}
  <ThisWeek week={home.week} />
{:else if home?.weekState === 'unavailable'}
  <ThisWeek />
{/if}
