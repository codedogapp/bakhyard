<script lang="ts">
import Hero from './home/Hero.svelte';
import { type Feed, pollHome } from './home/poll';
import Header from './shell/Header.svelte';
import TopBar from './shell/TopBar.svelte';

let feed = $state.raw<Feed>({ connection: 'connecting' });

$effect(() =>
  pollHome((next) => {
    feed = next;
  })
);
</script>

<TopBar connection={feed.connection} asOf={feed.asOf} />
<Header path={location.pathname} />
<main>
  <Hero home={feed.home} />
</main>
