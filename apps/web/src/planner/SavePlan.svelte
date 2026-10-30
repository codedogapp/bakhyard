<script lang="ts">
import { saveLabel } from './labels';
import { SaveState } from './save.svelte';

// The plan, by its ID as its page's path holds it, whose plan the button saves in the vault.
let { id }: { id: string } = $props();

const saving = new SaveState();
</script>

<!--
  Above the plan, as a note's tools are above the note. aria-disabled, not disabled, so the focus stays on the button
  while it saves.
-->
<div class="tools form-row">
  <button class="form-button" type="button" aria-disabled={saving.busy} onclick={() => void saving.save(id)}>
    Save to Obsidian
  </button>
  <p class="form-status" role="status">{saveLabel(saving.status)}</p>
</div>

<style>
.tools {
  margin-bottom: 30px;
}

/* A note's name can hold a long word, which breaks rather than widen the page. */
.form-status {
  overflow-wrap: anywhere;
}

@media (max-width: 700px) {
  .tools {
    margin-bottom: 24px;
  }
}
</style>
