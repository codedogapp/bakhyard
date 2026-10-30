<script lang="ts">
import { onMount } from 'svelte';

import PageHead from '../shell/PageHead.svelte';
import PolledSection from '../shell/PolledSection.svelte';
import { foundGames, type Games, loadGames, platformsLabel, playedLabel, playtimeLabel, trophiesLabel } from './games';

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
      <div class="list-row">
        <time class="list-label" datetime={game.lastPlayed}>{playedLabel(game.lastPlayed, now)}</time>
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
