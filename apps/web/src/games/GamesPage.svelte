<script lang="ts">
import { onMount } from 'svelte';

import { dayLabel } from '../shell/day';
import PageHead from '../shell/PageHead.svelte';
import PolledSection from '../shell/PolledSection.svelte';
import { coverImage, foundGames, type Games, loadGames, platformsLabel, playtimeLabel, trophiesLabel } from './games';

// The games I played lately (decision 307), loaded each time the page shows; now tells which day is today.
let { now }: { now: Date } = $props();

let games = $state.raw<Games>({ kind: 'loading' });

onMount(() => {
  void loadGames().then((loaded) => {
    games = loaded;
  });
});
</script>

<PageHead>Games</PageHead>
{#if games.kind === 'off'}
  <p class="page-status">Exophase isn't set up.</p>
{:else}
  <PolledSection
    polled={games}
    list={foundGames}
    title="Recently"
    accent="played"
    noun="game"
    empty="No games played in the last 14 days."
    failed="Couldn't load the games."
    markStale
  >
    {#snippet row(game)}
      <div class="list-row game">
        <time class="list-label" datetime={game.lastPlayed}>{dayLabel(game.lastPlayed, now)}</time>
        <!-- A background rather than an image: screen readers skip it, since the title names the game, and a cover that fails to load leaves the blank. -->
        <span class="cover" style:background-image={coverImage(game.cover)}></span>
        <span class="list-text">
          <span class="list-title">{game.title}</span>
          <span class="list-detail">{platformsLabel(game.platforms)}</span>
          {#if game.trophies}
            <span class="list-detail">{trophiesLabel(game.trophies)}</span>
          {/if}
        </span>
        <span class="list-end">
          {#if game.playtimeMinutes}
            {playtimeLabel(game.playtimeMinutes)}
          {/if}
        </span>
      </div>
    {/snippet}
  </PolledSection>
{/if}

<style>
/* The cover gets a column between the day and the title (decision 308). */
.game {
  grid-template-areas: "label cover text end";
  grid-template-columns: 120px 48px 1fr auto;
}

/* IGDB's 264 by 374 px cover, scaled to the height of a title over two detail lines; a blank when the game has none. */
.cover {
  grid-area: cover;
  width: 48px;
  height: 68px;
  border: 1px solid var(--rule);
  background: var(--fill) center / cover;
}

@media (max-width: 700px) {
  .game {
    grid-template-areas:
      "label label end"
      "cover text text";
    grid-template-columns: 48px 1fr auto;
  }
}
</style>
