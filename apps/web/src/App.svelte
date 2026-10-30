<script lang="ts">
import { tick } from 'svelte';

import GamesPage from './games/GamesPage.svelte';
import HomePage from './home/HomePage.svelte';
import { type Feed, pollHome } from './home/poll';
import { unavailableSources } from './home/sources';
import NotePage from './notes/NotePage.svelte';
import NotesPage from './notes/NotesPage.svelte';
import { NotesState } from './notes/state.svelte';
import NudgePage from './nudge/NudgePage.svelte';
import AppPage from './pages/AppPage.svelte';
import MorePage from './pages/MorePage.svelte';
import NotFound from './pages/NotFound.svelte';
import SignIn from './pages/SignIn.svelte';
import PlannerPage from './planner/PlannerPage.svelte';
import PullsPage from './pulls/PullsPage.svelte';
import { startClock } from './shell/clock';
import Header from './shell/Header.svelte';
import { followNavigation } from './shell/navigation';
import { pageAt, pageTitle, signedOutTitle } from './shell/page';
import TabBar from './shell/TabBar.svelte';
import TopBar from './shell/TopBar.svelte';
import TimePage from './time/TimePage.svelte';
import TodosPage from './todos/TodosPage.svelte';

let path = $state(location.pathname);
const page = $derived(pageAt(path));
let feed = $state.raw<Feed>({ connection: 'connecting' });
const signedOut = $derived(feed.connection === 'signed-out');
// The Notes page's state lives as long as the app, so the page is as it was left when the user comes back to it.
const notes = new NotesState();
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

// The new page is in the DOM before the browser resets focus and the scroll position for it.
$effect(() =>
  followNavigation((next) => {
    path = next;
    return tick();
  })
);
</script>

<svelte:head>
  <title>{signedOut ? signedOutTitle : pageTitle(page)}</title>
</svelte:head>

<TopBar {now} connection={feed.connection} asOf={feed.asOf} unavailable={unavailableSources(feed.home)} />
{#if signedOut}
  <main>
    <SignIn />
  </main>
{:else}
  <Header {path} />
  <main>
    {#if page.kind === 'home'}
      <HomePage {now} home={feed.home} />
    {:else if page.kind === 'notes'}
      <NotesPage {notes} />
    {:else if page.kind === 'note'}
      <NotePage note={page.note} {notes} />
    {:else if page.kind === 'todos'}
      <TodosPage />
    {:else if page.kind === 'pulls'}
      <PullsPage />
    {:else if page.kind === 'time'}
      <TimePage />
    {:else if page.kind === 'games'}
      <GamesPage {now} />
    {:else if page.kind === 'planner'}
      <PlannerPage {now} />
    {:else if page.kind === 'nudge'}
      <NudgePage />
    {:else if page.kind === 'app'}
      <AppPage app={page.app} />
    {:else if page.kind === 'more'}
      <MorePage items={feed.home?.items} />
    {:else}
      <NotFound {path} />
    {/if}
  </main>
  <TabBar {path} />
{/if}
