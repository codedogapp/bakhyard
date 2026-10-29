<script lang="ts">
import { onMount } from 'svelte';

import PageHead from '../shell/PageHead.svelte';
import Section from '../shell/Section.svelte';
import { searchedLabel } from '../shell/snapshot';
import DayList from './DayList.svelte';
import { missingNote } from './days';
import FillDialog from './FillDialog.svelte';
import FillForm from './FillForm.svelte';
import { Fill } from './fill.svelte';
import { loadReport, type TimeReport } from './report';
import WeekCard from './WeekCard.svelte';

let report = $state.raw<TimeReport>({ kind: 'loading' });
const fill = new Fill((loaded) => {
  report = loaded;
});

// The report loads each time the page shows, as the time service's latest poll has it (decision 184), and so do my
// open tickets, which the form offers.
onMount(() => {
  void loadReport().then((loaded) => {
    report = loaded;
  });
  void fill.load();
});
</script>

<PageHead>Time</PageHead>
{#if report.kind === 'off'}
  <p class="off">Jira isn't set up.</p>
{:else if report.kind === 'failed'}
  <p class="off">Couldn't load the time report.</p>
{:else if report.kind === 'loaded'}
  {@const { days, week, searched, failure } = report.snapshot}
  {#if failure}
    <p class={days ? 'failure' : 'off'}>
      The last search failed: {failure}.
      {#if days && searched}
        These hours are from <time datetime={searched}>{searchedLabel(searched)}</time>.
      {/if}
    </p>
  {/if}
  {#if days?.length}
    <FillForm {fill} {days} />
  {/if}
  {#if week}
    <Section title="This" accent="week">
      <WeekCard {week} />
    </Section>
  {/if}
  {#if days}
    <Section title="Last 7" accent="days" note={missingNote(days)}>
      <DayList {days} />
    </Section>
  {/if}
{/if}
<FillDialog {fill} />

<style>
.off,
.failure {
  margin: 0;
  padding: 0 var(--page-padding) 64px;
  color: var(--ink-muted);
  overflow-wrap: anywhere;
}

/* Over the sections, the line leaves the space that the Todos page's form leaves over its first section. */
.failure {
  padding-bottom: 30px;
}

@media (max-width: 700px) {
  .off {
    padding-bottom: 48px;
  }

  .failure {
    padding-bottom: 24px;
  }
}
</style>
