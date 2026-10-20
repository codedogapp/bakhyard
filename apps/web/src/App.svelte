<script lang="ts">
import Hero from './home/Hero.svelte';
import NeedsYou from './home/NeedsYou.svelte';
import { type Feed, pollHome } from './home/poll';
import ThisWeek from './home/ThisWeek.svelte';
import YourApps from './home/YourApps.svelte';
import { startClock } from './shell/clock';
import Header from './shell/Header.svelte';
import TabBar from './shell/TabBar.svelte';
import TopBar from './shell/TopBar.svelte';

const path = location.pathname;
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
<Header {path} />
<main>
  <Hero {now} home={feed.home} />
  {#if feed.home}
    <NeedsYou {now} items={feed.home.items} />
  {/if}
  <YourApps items={feed.home?.items} />
  {#if feed.home}
    <ThisWeek week={feed.home.week} />
  {/if}
</main>
<TabBar {path} />
