<script lang="ts">
import Properties from './Properties.svelte';
import type { RenderedNote } from './render';

// A note's Properties block and its text.
let { note }: { note: RenderedNote } = $props();
</script>

<article>
  <Properties properties={note.properties} />
  <!-- Sanitized HTML, with no raw HTML from the note (render.ts). -->
  <div class="note markdown">{@html note.html}</div>
</article>

<style>
/* The note's own elements, which Svelte can't scope; markdown.css styles the rest. */
.note :global {
  /* A block of math, spaced as the note's other blocks. A wide one scrolls sideways on its own, as KaTeX's docs show;
     2 px above and below keep the clip off the tips of tall glyphs, which reach a little past KaTeX's box. */
  .katex-display {
    margin: 0 0 16px;
    padding-block: 2px;
    overflow: auto hidden;
  }

  /* No space below the note's last block (markdown.css), math too. */
  > .katex-display:last-child {
    margin-bottom: 0;
  }

  /* A wikilink that leads nowhere: its text, dimmed, and not a link. */
  .unresolved {
    color: var(--ink-muted);
  }
}
</style>
