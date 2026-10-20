<script lang="ts">
import type { Item } from './api';
import { countNote, dueLabel, itemHref } from './items';
import Section from './Section.svelte';

let { items, now }: { items: readonly Item[]; now: Date } = $props();
</script>

<Section title="Needs you" accent="today" note={countNote(items.length)}>
  <div class="card">
    {#if items.length > 0}
      <ul>
        {#each items as item (item.id)}
          <li>
            <a href={itemHref(item)}>
              <span class="source">{item.source}</span>
              <span class="text">
                <span class="title">{item.title}</span>
                {#if item.detail}
                  <span class="detail">{item.detail}</span>
                {/if}
              </span>
              <span class="when">
                {#if item.late}
                  <span class="chip">Late</span>
                {/if}
                {#if item.due}
                  <time datetime={item.due}>{dueLabel(item.due, now)}</time>
                {/if}
              </span>
            </a>
          </li>
        {/each}
      </ul>
    {:else}
      <p class="empty">Nothing needs you today.</p>
    {/if}
  </div>
</Section>

<style>
.card {
  border: 1px solid var(--rule);
  background: var(--surface);
  box-shadow: var(--offset) var(--offset) 0 var(--shadow);
}

ul {
  margin: 0;
  padding: 0;
  list-style: none;
}

li + li {
  border-top: 1px solid var(--rule);
}

a {
  display: grid;
  grid-template-areas: "source text when";
  grid-template-columns: 120px 1fr auto;
  align-items: center;
  gap: 6px 20px;
  min-height: 64px;
  padding: 10px 20px;
  color: inherit;
  text-decoration: none;
}

a:hover {
  background: var(--fill);
}

.source,
.when {
  color: var(--ink-muted);
  font: 400 11px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
}

.source {
  grid-area: source;
}

.text {
  display: flex;
  flex-direction: column;
  grid-area: text;
  min-width: 0;
  overflow-wrap: anywhere;
}

.title {
  font-weight: 500;
}

.detail {
  color: var(--ink-muted);
  font-size: 14px;
}

.when {
  display: flex;
  align-items: center;
  gap: 10px;
  grid-area: when;
  white-space: nowrap;
}

.chip {
  padding: 3px 6px;
  border: 1px solid var(--ink);
  background: var(--ink);
  color: var(--paper);
  font-size: 10px;
  line-height: 1.2;
}

.empty {
  margin: 0;
  padding: 20px;
  color: var(--ink-muted);
}

@media (max-width: 700px) {
  a {
    grid-template-areas:
      "source when"
      "text text";
    grid-template-columns: 1fr auto;
  }
}
</style>
