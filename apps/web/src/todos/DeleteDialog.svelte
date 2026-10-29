<script lang="ts">
import type { Editor } from './editor.svelte';

let { editor }: { editor: Editor } = $props();

const id = $props.id();
</script>

<!--
  Opened as a modal, so the page behind it can't be reached and Escape closes it; as it closes, the browser gives the
  focus back to where it was before.
-->
<dialog
  class="dialog"
  role="alertdialog"
  aria-labelledby="{id}-title"
  aria-describedby="{id}-text"
  {@attach editor.dialog}
>
  <h2 class="dialog-title" id="{id}-title">Delete this todo?</h2>
  <p class="dialog-text" id="{id}-text">“{editor.todo?.title}” will be deleted. This can't be undone.</p>
  <!--
    The focus starts on the choice that changes nothing, since a delete can't be undone. Cancel comes first, so the
    browser would pick it anyway; autofocus keeps it so if the dialog ever gains a link or a field.
  -->
  <button class="form-button" type="button" autofocus onclick={() => editor.cancelDelete()}>Cancel</button>
  <button class="form-button form-button-primary" type="button" onclick={() => editor.delete()}>Delete</button>
</dialog>
