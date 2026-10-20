<script lang="ts">
import type { Connection } from '../home/poll';
import { clockLabel, startClock } from './clock';
import { statusLabel } from './status';

let { connection, asOf }: { connection: Connection; asOf?: Date } = $props();

let now = $state(new Date());

$effect(() =>
  startClock((date) => {
    now = date;
  })
);
</script>

<div class="top-bar">
  <span role="status">{statusLabel(connection, asOf)}</span>
  <span>{clockLabel(now)}</span>
</div>

<style>
.top-bar {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 8px var(--page-padding);
  background: var(--ink);
  color: var(--paper);
  font: 400 10px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
}
</style>
