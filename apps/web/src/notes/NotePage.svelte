<script lang="ts">
import PageHead from '../shell/PageHead.svelte';
import { fetchNote } from './api';
import NoteBody from './NoteBody.svelte';
import type { RenderedNote } from './render';
import { type Note, withoutExtension } from './tree';

// A note's page: its head at once, from its path; the note under it once it loads, each time the page shows it.
let { note }: { note: Note } = $props();

type Body = { kind: 'loading' | 'missing' | 'failed' } | { kind: 'loaded'; note: RenderedNote };

let body = $state.raw<Body>({ kind: 'loading' });

$effect(() => {
  // Going straight to another note's page drops this one's answer, should it come after.
  let current = true;
  body = { kind: 'loading' };
  void load(note).then((loaded) => {
    if (current) {
      body = loaded;
    }
  });
  return () => {
    current = false;
  };
});

// The note's file, and the renderer, a chunk of its own, the first time a note shows.
async function load(shown: Note): Promise<Body> {
  try {
    const [markdown, { renderNote }] = await Promise.all([fetchNote(shown.path), import('./render')]);
    return markdown === undefined ? { kind: 'missing' } : { kind: 'loaded', note: renderNote(markdown, shown.title) };
  } catch {
    return { kind: 'failed' };
  }
}
</script>

<div class="note-page">
  <PageHead eyebrow={note.folder}>{note.title}</PageHead>
  <div class="content">
    {#if body.kind === 'loaded'}
      <NoteBody note={body.note} />
    {:else if body.kind === 'missing'}
      <p class="failed">There's no note at {withoutExtension(note.path)}.</p>
    {:else if body.kind === 'failed'}
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

.failed {
  margin: 0;
  color: var(--ink-muted);
}

@media (max-width: 700px) {
  .content {
    padding-bottom: 48px;
  }
}
</style>
