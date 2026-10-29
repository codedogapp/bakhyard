<script lang="ts">
import { pushStatus } from './labels';
import type { Push } from './push';

type Off = Extract<Push, { kind: 'off' }>;

let { push, changed }: { push?: Push; changed: () => void } = $props();

// A failure belongs to the check it came from: once the bell checks again, the panel says what it found.
let failure = $state.raw<{ of: Off; reason: string }>();
let working = $state(false);

const status = $derived(failure && failure.of === push ? { kind: 'failed' as const, reason: failure.reason } : push);

// The tap's handler calls enable first, since Safari allows the permission prompt only within the tap. A second tap
// while it runs does nothing.
async function enable(off: Off) {
  if (working) {
    return;
  }
  working = true;
  const reason = await off.enable();
  working = false;
  if (reason) {
    failure = { of: off, reason };
  } else {
    changed();
  }
}
</script>

<div class="form-row">
  {#if push?.kind === 'off'}
    {@const off = push}
    <button type="button" class="form-button" aria-disabled={working} onclick={() => enable(off)}>
      Turn on notifications
    </button>
  {/if}
  <p class="form-status status" role="status">{pushStatus(status)}</p>
</div>

<style>
/* Alone in the panel, the line needs no space above it. */
.status:first-child {
  margin-top: 0;
}
</style>
