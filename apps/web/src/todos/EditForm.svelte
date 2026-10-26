<script lang="ts">
import type { Editor } from './editor.svelte';
import { focusField } from './focus';
import { editLabel } from './labels';
import PriorityPicker from './PriorityPicker.svelte';

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
    <!--
      The priority and the buttons share the field's line on a desktop. On a narrower screen they wrap under it, each on
      a line of its own on a phone, and Delete under the other buttons on the narrowest. The Move buttons come last, so
      they wrap first.
    -->
    <PriorityPicker {editor} />
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
    <!--
      Each moves the todo at once, without Save. aria-disabled, not disabled, so the focus stays on a button that has
      taken the todo to its group's end, or whose move runs.
    -->
    <span class="buttons">
      <button
        class="form-button"
        type="button"
        aria-disabled={editor.busy || !editor.canMove('up')}
        onclick={(event) => void editor.move('up', event.currentTarget)}
      >
        Move up
      </button>
      <button
        class="form-button"
        type="button"
        aria-disabled={editor.busy || !editor.canMove('down')}
        onclick={(event) => void editor.move('down', event.currentTarget)}
      >
        Move down
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
