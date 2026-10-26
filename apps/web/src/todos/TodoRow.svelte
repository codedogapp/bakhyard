<script lang="ts">
import type { Todo } from './api';
import { tickLabel } from './labels';
import type { TodosState } from './state.svelte';

let { list, todo }: { list: TodosState; todo: Todo } = $props();

function click(event: MouseEvent & { currentTarget: HTMLInputElement }): void {
  if (list.ticking(todo.id)) {
    event.preventDefault();
    return;
  }
  void list.tick(todo.id, event.currentTarget.checked);
}
</script>

<!-- The title labels the box, so a tap anywhere on it ticks the todo. -->
<li class={{ done: todo.done }}>
  <label>
    <!-- aria-disabled, not disabled, so focus stays on the box while its tick runs. -->
    <input type="checkbox" checked={todo.done} aria-disabled={list.ticking(todo.id)} onclick={click}>
    <span>{todo.title}</span>
  </label>
  <p class="status" role="status">{tickLabel(list.ticks.get(todo.id))}</p>
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
</style>
