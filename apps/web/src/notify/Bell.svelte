<script lang="ts">
import { onMount, tick } from 'svelte';

import { bellName } from './labels';
import Notifications from './Notifications.svelte';
import { type Push, preparePush } from './push';

const id = $props.id();
const panel = `${id}-panel`;
const gap = `${id}-gap`;

let push = $state.raw<Push>();
// Slashed once the page knows that notifications don't reach this device.
const slashed = $derived(push !== undefined && push.kind !== 'on');
let bell: HTMLButtonElement | undefined;
// Only the latest check counts, so a check that ends late can't undo a later one.
let checks = 0;

async function check() {
  checks += 1;
  const mine = checks;
  const next = await preparePush();
  if (mine === checks) {
    push = next;
  }
}

// Turning notifications on takes the panel's button away, and the bell gets the focus if it went with the button.
async function enabled() {
  await check();
  await tick();
  if (document.activeElement === document.body) {
    bell?.focus();
  }
}

// Opening the panel checks again, in case the browser's settings changed since; it says what it knew until then.
function toggled(event: ToggleEvent) {
  if (event.newState === 'open') {
    void check();
  }
}

onMount(() => {
  void check();
});
</script>

<button bind:this={bell} class="icon-button" type="button" popovertarget={panel} aria-label={bellName(push)}>
  <svg aria-hidden="true" viewBox="0 0 16 16" width="16" height="16">
    {#if slashed}
      <!-- The slash cuts a gap in the bell, so the two read apart at 16 px. -->
      <mask id={gap}>
        <rect width="16" height="16" fill="white" />
        <path d="M2 2l12 12" stroke="black" stroke-width="3.5" />
      </mask>
      <path d="M2 2l12 12" />
    {/if}
    <path
      d="M4 11.5V6.75a4 4 0 0 1 8 0v4.75M2 11.5h12M6.5 14.25h3M8 1.5v1.25"
      mask={slashed ? `url(#${gap})` : undefined}
    />
  </svg>
</button>
<!-- A popover, so a tap outside it or Escape closes it, and it opens over the page in the browser's top layer. -->
<div id={panel} class="panel" popover="auto" ontoggle={toggled}>
  <Notifications {push} changed={enabled} />
</div>

<style>
/* A card that hangs under the header at its right end (decision 179). */
.panel {
  position: absolute;
  inset: auto;
  top: anchor(--header bottom);
  right: var(--page-padding);
  max-width: min(360px, 100% - 2 * var(--page-padding));
  margin: 12px 0 0;
  padding: 20px;
  border: 1px solid var(--rule);
  background: var(--surface);
  color: var(--ink);
  box-shadow: var(--offset) var(--offset) 0 var(--shadow);
}
</style>
