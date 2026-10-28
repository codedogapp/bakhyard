<script lang="ts">
import type { Push } from './push';

let { push, changed }: { push?: Push; changed: () => void } = $props();

let failure = $state('');
let working = $state(false);

const says = $derived.by(() => {
  switch (push?.kind) {
    case 'on':
      return 'Notifications on';
    case 'blocked':
      return 'Notifications are blocked in the browser settings';
    case 'unsupported':
      return 'Notifications need the Home Screen app';
    case 'failed':
      return `Notifications failed: ${push.reason}`;
    default:
      return '';
  }
});

// The tap's handler calls enable first, since Safari allows the permission prompt only within the tap.
async function enable(turnOn: () => Promise<string | undefined>) {
  working = true;
  failure = (await turnOn()) ?? '';
  working = false;
  if (!failure) {
    changed();
  }
}
</script>

<div class="notifications form-row">
  {#if push?.kind === 'off'}
    {@const turnOn = push.enable}
    <button type="button" class="form-button" aria-disabled={working} onclick={() => enable(turnOn)}>
      Turn on notifications
    </button>
  {/if}
  <p class="form-status" role="status">{failure ? `Notifications failed: ${failure}` : says}</p>
</div>

<style>
.notifications {
  padding: 0 var(--page-padding) 40px;
}
</style>
