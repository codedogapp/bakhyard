<script lang="ts">
import Arrow from '../shell/Arrow.svelte';
import { type App, appHref } from '../shell/apps';
import type { Item } from './api';
import { itemCount, itemNoun } from './items';

// No items until the Home first loads: the cards still link to the apps, without counts. The names are headings one
// level under the heading the grid sits under.
let { apps, items, headingLevel }: { apps: readonly App[]; items?: readonly Item[]; headingLevel: 2 | 3 } = $props();
</script>

<ul class="grid">
  {#each apps as app (app.id)}
    <li>
      <a class="card" href={appHref(app.id)}>
        <span class="top">
          <span class="number" aria-hidden="true">{app.number}</span>
          <Arrow />
        </span>
        {#if items}
          {@const count = itemCount(items, app.id)}
          <span class="count" class:zero={count === 0}>
            <span class="value">{count}</span>
            <span class="unit">{itemNoun(count)}</span>
          </span>
        {/if}
        <svelte:element this={`h${headingLevel}`} class="name">{app.name}</svelte:element>
        <span class="description">{app.description}</span>
      </a>
    </li>
  {/each}
</ul>

<style>
.grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 24px;
  margin: 0;
  padding: 0;
  list-style: none;
}

li {
  display: flex;
}

.card {
  display: flex;
  flex: 1;
  flex-direction: column;
  padding: 20px;
  border: 1px solid var(--rule);
  background: var(--surface);
  box-shadow: var(--offset) var(--offset) 0 var(--shadow);
  color: inherit;
  text-decoration: none;
}

@media (hover: hover) {
  .card:hover {
    border-color: var(--ink);
    box-shadow: var(--offset-hover) var(--offset-hover) 0 var(--shadow-strong);
    transform: translateY(-2px);
  }
}

/* Like a key going down: the card moves towards its shadow, which shrinks by as much. */
.card:active {
  box-shadow: calc(var(--offset) - 2px) calc(var(--offset) - 2px) 0 var(--shadow-strong);
  transform: translate(2px, 2px);
}

@media (prefers-reduced-motion: no-preference) {
  .card {
    transition:
      transform 150ms ease,
      box-shadow 150ms ease,
      border-color 150ms ease;
  }
}

.top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.number {
  color: var(--ink-muted);
  font: 400 11px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
}

.count {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-top: 32px;
}

.value {
  font-size: 28px;
  font-weight: 500;
  letter-spacing: -0.03em;
  line-height: 1.2;
}

.zero .value,
.unit,
.description {
  color: var(--ink-muted);
}

.unit {
  font: 400 11px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
}

.name {
  margin: 16px 0 0;
  font-size: 19px;
  font-weight: 500;
  letter-spacing: -0.025em;
  line-height: 1.3;
}

.description {
  margin-top: 4px;
  font-size: 13px;
  line-height: 1.6;
}

@media (max-width: 1100px) {
  .grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 700px) {
  .grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
  }

  .value {
    font-size: 24px;
  }

  .name {
    font-size: 16px;
  }
}
</style>
