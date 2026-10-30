<script lang="ts">
import { onMount } from 'svelte';

import PageHead from '../shell/PageHead.svelte';
import AlertsSection from './AlertsSection.svelte';
import { type Alerts, loadAlerts } from './alerts';

// The alerts, loaded each time the page shows, as the PRs page's are (decision 606); now tells which day is today.
let { now }: { now: Date } = $props();

let list = $state.raw<Alerts>({ kind: 'loading' });

onMount(() => {
  void loadAlerts().then((loaded) => {
    list = loaded;
  });
});
</script>

<PageHead summary="The laptop's alerts: those that fire, and those of the last 30 days.">Alerts</PageHead>
{#if list.kind === 'failed'}
  <p class="page-status">Couldn't load the alerts.</p>
{:else if list.kind === 'loaded'}
  <AlertsSection alerts={list.firing} title="Firing" accent="now" empty="No alerts are firing." {now} />
  <AlertsSection
    alerts={list.resolved}
    title="The last"
    accent="30 days"
    empty="No alerts in the last 30 days."
    {now}
  />
{/if}
