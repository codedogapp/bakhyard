<script lang="ts">
import { startClock } from '../shell/clock';
import type { Home } from './api';
import { dateLabel, greeting, summary } from './greeting';

let { home }: { home?: Home } = $props();

let now = $state(new Date());

$effect(() =>
  startClock((date) => {
    now = date;
  })
);
</script>

<section class="hero">
  <div>
    <p class="date">{dateLabel(now)}</p>
    <!-- The punctuation sits inside the braces, so Svelte adds no `?? ''` fallback that no test can reach. -->
    {#if home}
      <h1>{`${greeting(now)}, `}<em>{home.name}</em>.</h1>
    {:else}
      <h1>{`${greeting(now)}.`}</h1>
    {/if}
  </div>
  {#if home}
    <p class="summary">{summary(home.items)}</p>
  {/if}
</section>

<style>
.hero {
  display: grid;
  grid-template-columns: 1.4fr 1fr;
  column-gap: 10%;
  row-gap: 20px;
  align-items: end;
  padding: 58px var(--page-padding) 50px;
}

.date {
  margin: 0 0 16px;
  color: var(--ink-muted);
  font: 400 11px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
}

h1 {
  margin: 0;
  font-size: clamp(38px, 4.5vw, 74px);
  font-weight: 500;
  letter-spacing: -0.045em;
  line-height: 1.04;
}

em {
  /* The italic capital leans into the comma; this restores a word space. */
  margin-inline-start: 0.06em;
  font-family: var(--font-serif);
  font-weight: 400;
  letter-spacing: 0;
}

.summary {
  max-width: 390px;
  margin: 0;
  color: var(--ink-muted);
  line-height: 1.65;
}

@media (max-width: 700px) {
  .hero {
    grid-template-columns: 1fr;
  }
}
</style>
