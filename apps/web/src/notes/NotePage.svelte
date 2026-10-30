<script lang="ts">
import PageHead from '../shell/PageHead.svelte';
import { saveLabel } from './labels';
import NoteBody from './NoteBody.svelte';
import { ShownNote } from './shown.svelte';
import type { NotesState } from './state.svelte';
import { type Note, withoutExtension } from './tree';

// A note's page: its head at once, from its path; the note under it once it loads, each time the page shows it. Its
// links need the vault's list, which App keeps.
let { note, notes }: { note: Note; notes: NotesState } = $props();

// A new one for each note, which drops the last one's edits and its answer, should it come after.
const shown = $derived(new ShownNote(note, notes));

$effect(() => {
  void shown.load();
});
</script>

<div class="note-page">
  <PageHead eyebrow={note.folder}>{note.title}</PageHead>
  <div class="content">
    {#if shown.body.kind === 'loaded'}
      {#if shown.editable || shown.history}
        <!--
          Save follows Cancel, so Cancel takes Edit's place, and Shift-Tab from the editor, which comes after them,
          reaches Save first, or, once a save has found the note changed, the offer to keep the text as a new note
          (decision 424), which wraps under them on a phone. aria-disabled, not disabled, so the focus stays on a
          button while its save runs. History, a link to the note's history in Gitea, comes last (decision 425).
        -->
        <div class="tools form-row">
          <span class="buttons">
            {#if shown.editable}
              <button
                class="form-button"
                type="button"
                aria-disabled={shown.busy}
                onclick={() => shown.toggle()}
                {@attach shown.button}
              >
                {shown.editing ? 'Cancel' : 'Edit'}
              </button>
            {/if}
            {#if shown.editing}
              <button class="form-button" type="button" aria-disabled={shown.busy} onclick={() => void shown.save()}>
                Save
              </button>
            {/if}
            {#if shown.stale}
              <button class="form-button" type="button" aria-disabled={shown.busy} onclick={() => void shown.keep()}>
                Save as new note
              </button>
            {/if}
            {#if shown.history}
              <a class="form-button history" href={shown.history}>History</a>
            {/if}
          </span>
          <p class="form-status" role="status">{saveLabel(shown.status)}</p>
        </div>
      {/if}
      {#if shown.editing}
        {#await import('./NoteEditor.svelte') then { default: NoteEditor }}
          <NoteEditor markdown={shown.body.markdown} page={shown} />
        {:catch}
          <p class="failed">Couldn't load the editor.</p>
        {/await}
      {:else}
        <NoteBody note={shown.body.note} />
      {/if}
    {:else if shown.body.kind === 'missing'}
      <p class="failed">There's no note at {withoutExtension(note.path)}.</p>
    {:else if shown.body.kind === 'failed'}
      <p class="failed">Couldn't load the note.</p>
    {/if}
  </div>
</div>

<style>
/* The head and the note share one column in the middle of the page: 700 px, Obsidian's readable line length, plus
   the gutters, which --page-padding gives inside the column. */
.note-page {
  max-width: calc(700px + 2 * (var(--gutter) + var(--inset-x)));
  margin-inline: auto;
}

.content {
  padding: 0 var(--page-padding) 64px;
}

.tools {
  margin-bottom: 30px;
}

.buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

/* A link that looks like the buttons beside it. */
.history {
  display: inline-flex;
  align-items: center;
  text-decoration: none;
}

.failed {
  margin: 0;
  color: var(--ink-muted);
}

@media (max-width: 700px) {
  .content {
    padding-bottom: 48px;
  }

  .tools {
    margin-bottom: 24px;
  }
}
</style>
