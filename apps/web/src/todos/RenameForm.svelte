<script lang="ts">
import { renameLabel } from './labels';
import { focusField, type Rename } from './rename.svelte';

let { rename }: { rename: Rename } = $props();

function submit(event: SubmitEvent): void {
  event.preventDefault();
  void rename.save();
}
</script>

<!-- Takes the place of the box and the title in the row: a form can't sit in the box's label. -->
<form onsubmit={submit}>
  <div class="form-row">
    <input
      class="form-field"
      type="text"
      aria-label="Title"
      readonly={rename.saving}
      bind:value={rename.draft}
      onkeydown={rename.keydown}
      {@attach focusField}
    >
    <!-- The buttons wrap together, under the field, on a narrow screen. -->
    <span class="buttons">
      <button class="form-button" type="submit" aria-disabled={rename.saving}>Save</button>
      <button class="form-button" type="button" aria-disabled={rename.saving} onclick={() => rename.close()}>
        Cancel
      </button>
    </span>
  </div>
  <p class="form-status" role="status">{renameLabel(rename.status)}</p>
</form>

<style>
form {
  flex: 1;
  min-width: 0;
  padding: 12px 20px;
}

.form-row {
  row-gap: 12px;
}

/* Wide enough to edit a title in: narrower, and the buttons wrap under it. */
input {
  flex-basis: 12em;
}

.buttons {
  display: flex;
  flex: none;
  gap: 12px;
}
</style>
