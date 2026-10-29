<script lang="ts">
import { dayRows } from './days';
import type { Day } from './report';

// The time report's days in a card, a row each: the date, the weekday, and the hours, after a Missing chip if missing.
let { days }: { days: Day[] } = $props();
</script>

<div class="card">
  <ul class="list">
    {#each dayRows(days) as row (row.date)}
      <li>
        <div class="list-row">
          <time class="list-label" datetime={row.date}>{row.label}</time>
          <span class="list-text">
            <span class="list-title">{row.title}</span>
          </span>
          <span class="list-end">
            {#if row.missing}
              <span class="chip chip-filled">Missing</span>
            {/if}
            <span class={{ logged: row.logged }}>{row.hours}</span>
          </span>
        </div>
      </li>
    {/each}
  </ul>
</div>

<style>
.card {
  border: 1px solid var(--rule);
  background: var(--surface);
  box-shadow: var(--offset) var(--offset) 0 var(--shadow);
}

/* The row's end is muted, so a day without hours stays quiet beside those with some. */
.logged {
  color: var(--ink);
}
</style>
