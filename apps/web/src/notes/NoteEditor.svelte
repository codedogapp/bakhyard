<script lang="ts">
import { noteEditor } from './editor';

// A note's Markdown in the editor, in place of the rendered note. A chunk of its own, with CodeMirror, which loads the
// first time Edit is pressed.
let { markdown }: { markdown: string } = $props();
</script>

<div class="editor" {@attach noteEditor(markdown)}></div>

<style>
/* CodeMirror's own elements, which Svelte can't scope: a form field's box (decision 112), as tall as the note, with
   the page's text in it. These win over CodeMirror's base theme, whose rules have one class fewer. */
.editor :global {
  .cm-editor {
    border: 1px solid var(--rule);
    background: var(--surface);
  }

  .cm-editor:hover {
    border-color: var(--ink);
  }

  /* The focus ring of the app's other controls, around the whole box rather than its text alone. */
  .cm-editor.cm-focused {
    outline: 2px solid var(--ink);
    outline-offset: 3px;
  }

  .cm-scroller {
    font-family: inherit;
    line-height: inherit;
  }

  /* Room to start a note that has no text yet. The caret is CodeMirror's black otherwise, which a dark page hides. */
  .cm-content {
    min-height: 240px;
    padding-block: 10px;
    caret-color: var(--ink);
  }

  .cm-line {
    padding-inline: 14px;
  }
}
</style>
