<script lang="ts">
import { appHref } from '../shell/apps';
import Section from '../shell/Section.svelte';
import WeekCard from '../time/WeekCard.svelte';
import type { Week } from '../time/week';

let { week }: { week?: Week } = $props();
</script>

<Section title="This" accent="week" link={{ href: appHref('time'), label: 'Day report' }}>
  <WeekCard {week} />
</Section>
