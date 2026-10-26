<script lang="ts">
import type { Editor } from './editor.svelte';

let { editor }: { editor: Editor } = $props();

const id = $props.id();
</script>

<!--
  Opened as a modal, so the page behind it can't be reached and Escape closes it; as it closes, the browser gives the
  focus back to where it was before.
-->
<dialog role="alertdialog" aria-labelledby="{id}-title" aria-describedby="{id}-text" {@attach editor.dialog}>
  <h2 id="{id}-title">Delete this todo?</h2>
  <p id="{id}-text">“{editor.todo?.title}” will be deleted. This can't be undone.</p>
  <!--
    The focus starts on the choice that changes nothing, since a delete can't be undone. Cancel comes first, so the
    browser would pick it anyway; autofocus keeps it so if the dialog ever gains a link or a field.
  -->
  <button class="form-button" type="button" autofocus onclick={() => editor.cancelDelete()}>Cancel</button>
  <button class="form-button primary" type="button" onclick={() => editor.delete()}>Delete</button>
</dialog>

<style>
dialog {
  width: min(400px, 100% - 2 * var(--gutter));
  padding: 20px;
  border: 1px solid var(--rule);
  background: var(--surface);
  color: var(--ink);
  box-shadow: var(--offset) var(--offset) 0 var(--shadow);
}

/* Only while open: a display of its own would show the closed dialog too. */
dialog[open] {
  display: flex;
  flex-wrap: wrap;
  column-gap: 12px;
}

/* A veil of the page's own colour over the page. */
dialog::backdrop {
  background: color-mix(in srgb, var(--paper) 75%, transparent);
}

h2,
p {
  flex-basis: 100%;
}

h2 {
  margin: 0;
  font-size: 19px;
  font-weight: 500;
  letter-spacing: -0.025em;
  line-height: 1.3;
}

/* A long word breaks rather than widen the dialog. */
p {
  margin: 8px 0 20px;
  overflow-wrap: anywhere;
}

.primary {
  background: var(--ink);
  color: var(--paper);
}

.primary:hover {
  background: var(--paper);
  color: var(--ink);
}

@media (max-width: 700px) {
  h2 {
    font-size: 16px;
  }
}
</style>
