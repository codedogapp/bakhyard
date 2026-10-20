<script lang="ts">
import Header from './shell/Header.svelte';
import TopBar from './shell/TopBar.svelte';
</script>

<TopBar />
<Header path={location.pathname} />
<main>
  <h1>Bakhyard</h1>
</main>
