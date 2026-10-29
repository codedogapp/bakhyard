<script lang="ts">
import { hoursLabel, type Week, weekPercent } from './week';

// Without a week, the card says the hours are unavailable in their place, with no label or bar: an empty bar
// would read as nothing logged (decision 187).
let { week }: { week?: Week } = $props();
</script>

<div class="card">
  {#if week}
    <p class="hours">
      <span class="value">{hoursLabel(week)}</span>
      <span class="unit">Logged on Jira</span>
    </p>
    <!-- The hours above say the same in words. The % sits inside the braces, so Svelte adds no `?? ''` fallback. -->
    <div class="track" aria-hidden="true">
      <div class="fill" style:width={`${weekPercent(week)}%`}></div>
    </div>
  {:else}
    <p class="hours"><span class="value unavailable">Hours unavailable</span></p>
  {/if}
</div>

<style>
.card {
  padding: 20px;
  border: 1px solid var(--rule);
  background: var(--surface);
  box-shadow: var(--offset) var(--offset) 0 var(--shadow);
}

.hours {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0 8px;
  margin: 0;
}

.value {
  font-size: 28px;
  font-weight: 500;
  letter-spacing: -0.03em;
  line-height: 1.2;
}

.unavailable {
  color: var(--ink-muted);
}

.unit {
  color: var(--ink-muted);
  font: 400 11px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
}

.track {
  height: 8px;
  margin-top: 16px;
  background: var(--fill);
}

.fill {
  height: 100%;
  background: var(--ink);
}

@media (max-width: 700px) {
  .value {
    font-size: 24px;
  }
}
</style>
