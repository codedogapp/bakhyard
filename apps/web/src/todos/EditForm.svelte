<script lang="ts">
import { type Editor, focusField } from './editor.svelte';
import { editLabel } from './labels';

let { editor }: { editor: Editor } = $props();

function submit(event: SubmitEvent): void {
  event.preventDefault();
  void editor.save();
}
</script>

<!-- Takes the place of the box and the title in the row: a form can't sit in the box's label. -->
<form onsubmit={submit}>
  <div class="form-row">
    <input
      class="form-field"
      type="text"
      aria-label="Title"
      readonly={editor.busy}
      bind:value={editor.draft}
      onkeydown={editor.keydown}
      {@attach focusField}
    >
    <!-- The buttons wrap under the field on a narrow screen, and Delete under the others on the narrowest. -->
    <span class="buttons">
      <button class="form-button" type="submit" aria-disabled={editor.busy}>Save</button>
      <button class="form-button" type="button" aria-disabled={editor.busy} onclick={() => editor.close()}>
        Cancel
      </button>
      <button
        class="form-button"
        type="button"
        aria-disabled={editor.busy}
        onclick={() => editor.confirmDelete()}
        {@attach editor.deleteButton}
      >
        Delete
      </button>
    </span>
  </div>
  <p class="form-status" role="status">{editLabel(editor.status)}</p>
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
  flex-wrap: wrap;
  gap: 12px;
}
</style>
