<script lang="ts">
import { countLabel } from '../shell/count';
import { dayLabel } from '../shell/day';
import Section from '../shell/Section.svelte';
import { type Alert, alertTitle, spanLabel } from './alerts';

// One of the Alerts page's lists, in a section under the head; empty says when it has no alert. Each row has the day
// the alert fired, what it says, and when it fired and resolved.
let {
  alerts,
  title,
  accent,
  empty,
  now,
}: { alerts: readonly Alert[]; title: string; accent: string; empty: string; now: Date } = $props();
</script>

<Section {title} {accent} note={countLabel(alerts.length, 'alert')}>
  <div class="card">
    {#if alerts.length > 0}
      <ul class="list">
        <!-- No key: an alert has no id, and the list is drawn anew with each load. -->
        {#each alerts as alert}
          <li class="list-row">
            <time class="list-label" datetime={alert.startedAt}>{dayLabel(alert.startedAt, now)}</time>
            <span class="list-text">
              <span class="list-title">{alertTitle(alert)}</span>
              {#if alert.description}
                <span class="list-detail">{alert.description}</span>
              {/if}
            </span>
            <span class="list-end">{spanLabel(alert)}</span>
          </li>
        {/each}
      </ul>
    {:else}
      <p class="empty">{empty}</p>
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
