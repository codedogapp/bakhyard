<script lang="ts">
import type { Connection } from '../home/poll';
import { clockLabel } from './clock';
import { statusLabel } from './status';

let {
  now,
  connection,
  asOf,
  unavailable = 0,
}: { now: Date; connection: Connection; asOf?: Date; unavailable?: number } = $props();
</script>

<div class="top-bar">
  <span role="status">{statusLabel(connection, asOf, unavailable)}</span>
  <span>{clockLabel(now)}</span>
</div>

<style>
.top-bar {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: calc(8px + env(safe-area-inset-top)) var(--page-padding) 8px;
  background: var(--ink);
  color: var(--paper);
  font: 400 10px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
}
</style>
