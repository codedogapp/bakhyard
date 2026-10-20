<script lang="ts">
import { clockLabel, startClock } from './clock';

let now = $state(new Date());

$effect(() =>
  startClock((date) => {
    now = date;
  })
);
</script>

<div class="top-bar">
  <span class="clock">{clockLabel(now)}</span>
</div>

<style>
.top-bar {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 8px var(--gutter);
  background: var(--ink);
  color: var(--paper);
  font: 400 10px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
}

.clock {
  margin-inline-start: auto;
}
</style>
