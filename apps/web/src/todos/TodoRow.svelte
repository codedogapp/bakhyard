<script lang="ts">
import type { Todo } from './api';
import { tickLabel } from './labels';
import RenameForm from './RenameForm.svelte';
import type { Rename } from './rename.svelte';
import type { TodosState } from './state.svelte';

let { list, todo, rename }: { list: TodosState; todo: Todo; rename: Rename } = $props();

const editing = $derived(rename.id === todo.id);

function click(event: MouseEvent & { currentTarget: HTMLInputElement }): void {
  if (list.ticking(todo.id)) {
    event.preventDefault();
    return;
  }
  void list.tick(todo.id, event.currentTarget.checked);
}
</script>

<li class={{ done: todo.done }}>
  {#if editing}
    <RenameForm {rename} />
  {:else}
    <!-- The title labels the box, so a tap anywhere on it ticks the todo. -->
    <label>
      <!-- aria-disabled, not disabled, so focus stays on the box while its tick runs. -->
      <input type="checkbox" checked={todo.done} aria-disabled={list.ticking(todo.id)} onclick={click}>
      <span>{todo.title}</span>
    </label>
  {/if}
  <p class="status" role="status">{tickLabel(list.ticks.get(todo.id))}</p>
  {#if !editing}
    <button
      class="edit"
      type="button"
      aria-label="Edit {todo.title}"
      aria-disabled={rename.saving}
      onclick={() => rename.open(todo)}
      {@attach rename.editButton(todo.id)}
    >
      Edit
    </button>
  {/if}
</li>

<style>
li {
  display: flex;
  align-items: flex-start;
}

li:not(:first-child) {
  border-top: 1px solid var(--rule);
}

@media (hover: hover) {
  li:hover {
    background: var(--fill);
  }

  .edit:hover {
    color: var(--ink);
  }
}

/* A long word breaks rather than widen the page. */
label {
  display: flex;
  flex: 1;
  gap: 14px;
  align-items: flex-start;
  min-width: 0;
  padding: 12px 20px;
  cursor: pointer;
  overflow-wrap: anywhere;
}

/* Drawn like a note's task box, and centred on the title's first line. */
input {
  flex: none;
  width: 18px;
  height: 18px;
  margin: 3px 0 0;
  padding: 3px;
  border: 1px solid var(--ink);
  border-radius: 0;
  background-clip: content-box;
  cursor: pointer;
  appearance: none;
}

input:checked {
  background-color: var(--ink);
}

.done span {
  color: var(--ink-muted);
  text-decoration: line-through;
}

/* Never hidden: a live region has to be there before its first change for screen readers to announce it. */
.status {
  flex: none;
  margin: 0;
  padding-block: 12px;
  color: var(--ink-muted);
  font: 400 11px / 24px var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
}

.status:not(:empty) {
  padding-right: 20px;
}

/* In the status line's style, and 48 px tall, as a tap target that takes in the 20 px gap before it. */
.edit {
  flex: none;
  margin-left: -20px;
  padding: 12px 20px;
  border: 0;
  background: none;
  color: var(--ink-muted);
  cursor: pointer;
  font: 400 11px / 24px var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
}

/* Inside the target, as a tab's, so it neither crosses the card's edge nor sits off-centre. */
.edit:focus-visible {
  outline-offset: -5px;
}
</style>
