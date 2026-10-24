<script lang="ts">
import type { SvelteSet } from 'svelte/reactivity';

import Arrow from '../home/Arrow.svelte';
import { countLabel } from './labels';
import Tree from './Tree.svelte';
import { type Folder, noteHref } from './tree';

// A folder's folders, each a row that opens to its own tree, then its notes. Rows are indented by their depth.
let { folder, open, depth = 0 }: { folder: Folder; open: SvelteSet<string>; depth?: number } = $props();

// The browser opens and closes folders; the set remembers which are open for the next time the tree shows. Not
// bind:open, which writes every closed folder back to the page whenever the set changes, closing one whose toggle
// event is still on its way.
function remember(event: Event & { currentTarget: HTMLDetailsElement }, path: string) {
  if (event.currentTarget.open) {
    open.add(path);
  } else {
    open.delete(path);
  }
}
</script>

<ul class:top={depth === 0} style:--depth={depth}>
  {#each folder.folders as child (child.path)}
    <li>
      <details open={open.has(child.path)} ontoggle={(event) => remember(event, child.path)}>
        <summary class="row">
          <Arrow />
          <span class="name">{child.name}</span>
          <span class="count">{countLabel(child.count)}</span>
        </summary>
        <Tree folder={child} {open} depth={depth + 1} />
      </details>
    </li>
  {/each}
  {#each folder.notes as note (note.path)}
    <li><a class="row" href={noteHref(note.path)}>{note.title}</a></li>
  {/each}
</ul>

<style>
ul {
  margin: 0;
  padding: 0;
  list-style: none;
}

/* Every row has a rule above it, but the card's first. */
li {
  border-top: 1px solid var(--rule);
}

.top > li:first-child {
  border-top: 0;
}

/* A child starts where its folder's name does: past the arrow and the gap. */
.row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 48px;
  padding: 10px 20px 10px calc(20px + var(--depth) * 28px);
  color: inherit;
  text-decoration: none;
  overflow-wrap: anywhere;
}

.row:hover {
  background: var(--fill);
}

summary {
  cursor: pointer;
  list-style: none;
}

summary::-webkit-details-marker {
  display: none;
}

summary > :global(svg) {
  flex: none;
}

details[open] > summary > :global(svg) {
  transform: rotate(90deg);
}

.name {
  flex: 1;
  font-weight: 500;
}

.count {
  color: var(--ink-muted);
  font: 400 11px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
  white-space: nowrap;
}
</style>
