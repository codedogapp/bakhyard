<script lang="ts">
import type { Editor } from './editor.svelte';
import { priorities } from './groups';
import { priorityLabel } from './labels';

let { editor }: { editor: Editor } = $props();

/** Keeps the priority while a save or a delete runs: a cancelled click leaves the radios as they were. */
function click(event: MouseEvent): void {
  if (editor.busy) {
    event.preventDefault();
  }
}
</script>

<!-- Named like the field, with no legend in sight: the editor shows no labels, and the section heads name the priorities. -->
<fieldset aria-label="Priority">
  {#each priorities as priority (priority)}
    <label>
      <!-- aria-disabled, not disabled, so the focus stays on the radio while a save runs. -->
      <input
        type="radio"
        name="priority"
        value={priority}
        aria-disabled={editor.busy}
        onclick={click}
        bind:group={editor.priority}
      >
      {priorityLabel(priority)}
    </label>
  {/each}
</fieldset>

<style>
/* No frame or spacing of its own: the options sit in the editor's row as its buttons do. */
fieldset {
  display: flex;
  flex-wrap: wrap;
  gap: 0 16px;
  margin: 0;
  padding: 0;
  border: 0;
}

/* As tall as the field and the buttons, so the whole label takes the tap. */
label {
  display: flex;
  gap: 8px;
  align-items: center;
  min-height: 44px;
  cursor: pointer;
}

/* Drawn like the row's tick box, but round, and in the label's colour. */
input {
  flex: none;
  width: 18px;
  height: 18px;
  margin: 0;
  padding: 3px;
  border: 1px solid currentColor;
  border-radius: 50%;
  color: inherit;
  background-clip: content-box;
  cursor: inherit;
  appearance: none;
}

input:checked {
  background-color: currentColor;
}

/* Muted while a save or a delete runs, as the buttons are. */
label:has([aria-disabled="true"]) {
  color: var(--ink-muted);
  cursor: default;
}
</style>
