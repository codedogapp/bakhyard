<script lang="ts">
import PageHead from '../shell/PageHead.svelte';

let { path }: { path: string } = $props();
</script>

<PageHead summary={`There's no page at ${path}.`}>Page not found</PageHead>
