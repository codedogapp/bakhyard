<script lang="ts">
import { type Note, noteHref } from './tree';

// The notes that match the filter, each with its folder, in the tree's order.
let { notes, filter }: { notes: readonly Note[]; filter: string } = $props();
</script>

{#if notes.length > 0}
  <ul>
    {#each notes as note (note.path)}
      <li>
        <a href={noteHref(note.path)}>
          <span class="title">{note.title}</span>
          {#if note.folder}
            <span class="folder">{note.folder}</span>
          {/if}
        </a>
      </li>
    {/each}
  </ul>
{:else}
  <p class="empty">No notes match “{filter.trim()}”.</p>
{/if}

<style>
ul {
  margin: 0;
  padding: 0;
  list-style: none;
}

li + li {
  border-top: 1px solid var(--rule);
}

a {
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-height: 48px;
  padding: 10px 20px;
  color: inherit;
  text-decoration: none;
  overflow-wrap: anywhere;
}

a:hover {
  background: var(--fill);
}

.title {
  font-weight: 500;
}

.folder {
  color: var(--ink-muted);
  font-size: 14px;
}

.empty {
  margin: 0;
  padding: 20px;
  color: var(--ink-muted);
}
</style>
