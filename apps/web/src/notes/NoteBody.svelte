<script lang="ts">
import Properties from './Properties.svelte';
import type { RenderedNote } from './render';

// A note's Properties block and its text, in a column as wide as Obsidian's readable line length.
let { note }: { note: RenderedNote } = $props();
</script>

<article>
  <Properties properties={note.properties} />
  <!-- Sanitized HTML, with no raw HTML from the note (render.ts). -->
  <div class="note">{@html note.html}</div>
</article>

<style>
article {
  max-width: 700px;
}

.note {
  overflow-wrap: break-word;
}

/* The note's own elements, which Svelte can't scope. */
.note :global {
  :is(p, ul, ol, pre, blockquote, table) {
    margin: 0 0 16px;
  }

  :is(h2, h3, h4, h5, h6) {
    margin: 32px 0 12px;
  }

  h2 {
    font-size: 28px;
    font-weight: 500;
    letter-spacing: -0.03em;
    line-height: 1.2;
  }

  h3 {
    font-size: 19px;
    font-weight: 500;
    letter-spacing: -0.025em;
    line-height: 1.3;
  }

  :is(h4, h5, h6) {
    font-size: 16px;
    font-weight: 600;
  }

  > :first-child {
    margin-top: 0;
  }

  > :last-child {
    margin-bottom: 0;
  }

  a {
    color: var(--ink);
    text-underline-offset: 2px;
  }

  :is(ul, ol) {
    padding-left: 24px;
  }

  li > :is(ul, ol) {
    margin: 0;
  }

  .task-list-item {
    list-style: none;
  }

  /* Drawn here, since browsers grey out a disabled checkbox; in the bullet's place, so a task's text lines up with the
     other items'. A done task's box has an ink square inside. */
  .task-list-item > input {
    appearance: none;
    box-sizing: border-box;
    width: 14px;
    height: 14px;
    margin: 0 7px 0 -21px;
    padding: 2px;
    border: 1px solid var(--ink);
    background-clip: content-box;
    vertical-align: -2px;
  }

  .task-list-item > input:checked {
    background-color: var(--ink);
  }

  blockquote {
    padding-left: 16px;
    border-left: 2px solid var(--ink);
    margin-inline: 0;
    color: var(--ink-muted);
  }

  hr {
    margin: 24px 0;
    border: 0;
    border-top: 1px solid var(--rule);
  }

  /* Code keeps its case, unlike the page's other mono text: a chord such as "Em" means what it says. */
  code {
    padding: 1px 4px;
    background: var(--fill);
    font-family: var(--font-mono);
    font-size: 0.875em;
  }

  /* A long line scrolls the block sideways, not the page. */
  pre {
    padding: 12px 16px;
    overflow-x: auto;
    background: var(--fill);
    font-size: 14px;
    line-height: 1.5;
  }

  pre code {
    padding: 0;
    font-size: inherit;
  }

  /* A wide table scrolls sideways on its own, as on GitHub. */
  table {
    display: block;
    width: max-content;
    max-width: 100%;
    overflow: auto;
    border-collapse: collapse;
  }

  :is(th, td) {
    padding: 6px 12px;
    border: 1px solid var(--rule);
  }

  th {
    font-weight: 600;
  }

  /* A column the note doesn't align starts at the left, as its cells do. */
  th:not([align]) {
    text-align: start;
  }
}

@media (max-width: 700px) {
  .note :global {
    h2 {
      font-size: 24px;
    }

    h3 {
      font-size: 16px;
    }
  }
}
</style>
