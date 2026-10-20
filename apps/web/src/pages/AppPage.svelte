<script lang="ts">
import type { App } from '../shell/apps';
import PageHead from '../shell/PageHead.svelte';

let { app }: { app: App } = $props();
</script>

<PageHead eyebrow="Coming soon" summary={app.description}>{app.name}</PageHead>
