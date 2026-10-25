<script lang="ts">
import type { Snippet } from 'svelte';

// A page's head: an eyebrow over its h1, with a summary beside them, or under them on phones. The h1 is the snippet.
let { eyebrow, summary, children }: { eyebrow?: string; summary?: string; children: Snippet } = $props();
</script>

<section class="page-head">
  <div>
    {#if eyebrow}
      <p class="eyebrow">{eyebrow}</p>
    {/if}
    <h1>{@render children()}</h1>
  </div>
  {#if summary}
    <p class="summary">{summary}</p>
  {/if}
</section>

<style>
.page-head {
  display: grid;
  grid-template-columns: 1fr;
  column-gap: 10%;
  row-gap: 20px;
  align-items: end;
  padding: 58px var(--page-padding) 50px;
  /* A long word, such as in a note's name, breaks rather than widen the page. */
  overflow-wrap: anywhere;
}

/* Without a summary, the eyebrow and the h1 take the whole width. */
.page-head:has(.summary) {
  grid-template-columns: 1.4fr 1fr;
}

.eyebrow {
  margin: 0 0 16px;
  color: var(--ink-muted);
  font: 400 11px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
}

h1 {
  margin: 0;
  font-size: clamp(38px, 4.5vw, 74px);
  font-weight: 500;
  letter-spacing: -0.045em;
  line-height: 1.04;
}

.summary {
  max-width: 390px;
  margin: 0;
  color: var(--ink-muted);
  line-height: 1.65;
}

@media (max-width: 700px) {
  .page-head:has(.summary) {
    grid-template-columns: 1fr;
  }
}
</style>
