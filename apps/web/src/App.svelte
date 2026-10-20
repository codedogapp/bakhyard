<script lang="ts">
import Hero from './home/Hero.svelte';
import NeedsYou from './home/NeedsYou.svelte';
import { type Feed, pollHome } from './home/poll';
import { startClock } from './shell/clock';
import Header from './shell/Header.svelte';
import TopBar from './shell/TopBar.svelte';

let feed = $state.raw<Feed>({ connection: 'connecting' });
// One clock for the page, so everything that shows the time moves at the same moment.
let now = $state.raw(new Date());

$effect(() =>
  pollHome((next) => {
    feed = next;
  })
);

$effect(() =>
  startClock((date) => {
    now = date;
  })
);
</script>

<TopBar {now} connection={feed.connection} asOf={feed.asOf} />
<Header path={location.pathname} />
<main>
  <Hero {now} home={feed.home} />
  {#if feed.home}
    <NeedsYou {now} items={feed.home.items} />
  {/if}
</main>
