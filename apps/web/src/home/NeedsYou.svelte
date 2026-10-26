<script lang="ts">
import Section from '../shell/Section.svelte';
import type { Item } from './api';
import { countNote, dueLabel, itemHref } from './items';

let { items, now }: { items: readonly Item[]; now: Date } = $props();
</script>

<Section title="Needs you" accent="today" note={countNote(items.length)}>
  <div class="card">
    {#if items.length > 0}
      <ul class="list">
        {#each items as item (item.id)}
          <li>
            <a class="list-row" href={itemHref(item)}>
              <span class="list-label">{item.source}</span>
              <span class="list-text">
                <span class="list-title">{item.title}</span>
                {#if item.detail}
                  <span class="list-detail">{item.detail}</span>
                {/if}
              </span>
              <span class="list-end">
                {#if item.late}
                  <span class="chip chip-filled">Late</span>
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

.empty {
  margin: 0;
  padding: 20px;
  color: var(--ink-muted);
}
</style>
