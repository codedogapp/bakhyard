<script lang="ts">
import { apps } from '../shell/apps';
import Section from '../shell/Section.svelte';
import AppCards from './AppCards.svelte';
import type { Item } from './api';

let { items }: { items?: readonly Item[] } = $props();
</script>

<Section title="Your" accent="apps">
  <AppCards {apps} {items} headingLevel={3} />
</Section>
