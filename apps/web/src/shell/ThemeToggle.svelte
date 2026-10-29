<script lang="ts">
import { applyChoice, loadChoice, nextChoice, saveChoice, type ThemeChoice } from './theme';

const names: Record<ThemeChoice, string> = { system: 'System', light: 'Light', dark: 'Dark' };

let choice = $state(loadChoice());

function cycle() {
  choice = nextChoice(choice);
  saveChoice(choice);
  applyChoice(choice);
}
</script>

<button class="icon-button" type="button" aria-label="Theme: {names[choice]}" onclick={cycle}>
  <svg aria-hidden="true" viewBox="0 0 16 16" width="16" height="16">
    {#if choice === 'system'}
      <!-- Half dark, half light: the OS decides. -->
      <circle cx="8" cy="8" r="6.25" />
      <path class="solid" d="M8 1.75a6.25 6.25 0 0 0 0 12.5z" />
    {:else if choice === 'light'}
      <circle cx="8" cy="8" r="2.75" />
      <path
        d="M13 8h1.5M11.54 11.54l1.06 1.06M8 13v1.5M4.46 11.54L3.4 12.6M3 8H1.5M4.46 4.46L3.4 3.4M8 3V1.5M11.54 4.46l1.06-1.06"
      />
    {:else}
      <path d="M7.53 1.77A6.25 6.25 0 1 0 14.23 8.47A5 5 0 0 1 7.53 1.77z" />
    {/if}
  </svg>
</button>

<style>
.solid {
  fill: currentColor;
  stroke: none;
}
</style>
