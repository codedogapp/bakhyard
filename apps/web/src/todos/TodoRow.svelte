<script lang="ts">
import type { Todo } from './api';
import type { Drag } from './drag.svelte';
import EditForm from './EditForm.svelte';
import type { Editor } from './editor.svelte';
import type { Focus } from './focus';
import { rowLabel } from './labels';
import type { TodosState } from './state.svelte';
import TicketBadge from './TicketBadge.svelte';
import type { Ticket } from './tickets';

// `open` holds my open tickets by key, which give a linked todo's badge its status and the editor its choices.
let {
  list,
  todo,
  editor,
  focus,
  drag,
  open,
}: {
  list: TodosState;
  todo: Todo;
  editor: Editor;
  focus: Focus;
  drag: Drag;
  open: ReadonlyMap<string, Ticket>;
} = $props();

const editing = $derived(editor.id === todo.id);

function click(event: MouseEvent & { currentTarget: HTMLInputElement }): void {
  if (list.ticking(todo.id)) {
    event.preventDefault();
    return;
  }
  void list.tick(todo.id, event.currentTarget.checked);
}
</script>

<li class={[{ done: todo.done }, drag.look(todo.id)]} {@attach drag.row(todo.id)}>
  {#if editing}
    <EditForm {editor} {open} />
  {:else}
    <!-- The title labels the box, so a tap anywhere on it ticks the todo. -->
    <label>
      <!-- aria-disabled, not disabled, so focus stays on the box while its tick runs. -->
      <input type="checkbox" checked={todo.done} aria-disabled={list.ticking(todo.id)} onclick={click}>
      <span class="text">
        <span class="title">{todo.title}</span>
        {#if todo.ticket}
          <TicketBadge key={todo.ticket} ticket={open.get(todo.ticket)} />
        {/if}
      </span>
    </label>
  {/if}
  <p class="status" role="status">{rowLabel(list.ticks.get(todo.id), list.dropFailed === todo.id)}</p>
  {#if !editing}
    <button
      class="edit"
      type="button"
      aria-label="Edit {todo.title}"
      aria-disabled={editor.busy}
      onclick={() => editor.open(todo)}
      {@attach focus.editButton(todo.id)}
    >
      Edit
    </button>
    <!-- For a pointer only: the editor's moves and priority do the same from the keyboard (decision 120). -->
    <span class={['grip', { waiting: drag.waiting }]} aria-hidden="true" {@attach drag.grip(todo.id)}>
      <svg aria-hidden="true" viewBox="0 0 16 16" width="16" height="16">
        <path d="M6 4h0M10 4h0M6 8h0M10 8h0M6 12h0M10 12h0" />
      </svg>
    </span>
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
  li:not(.lifted):hover {
    background: var(--fill);
  }

  .edit:hover,
  .grip:hover {
    color: var(--ink);
  }
}

/* Lifted like a hovered card, the dragged row stays in place, so it hides neither the line nor another row. */
.lifted {
  outline: 1px solid var(--ink);
  box-shadow: var(--offset-hover) var(--offset-hover) 0 var(--shadow-strong);
}

/* Where the dragged todo would land: a 2 px line across the top of the row it would go before, or the bottom. */
.drop-before {
  box-shadow: inset 0 2px var(--ink);
}

.drop-after {
  box-shadow: inset 0 -2px var(--ink);
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

/* The title over the ticket's badge, if the todo names one. */
.text {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

/* The title alone: the ticket's state is its own. */
.done .title {
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

/* 44 px wide and the row's height, with the dots on the title's first line; a finger on it drags, never scrolls. */
.grip {
  flex: none;
  align-self: stretch;
  padding: 16px 14px 0;
  color: var(--ink-muted);
  cursor: grab;
  touch-action: none;
  -webkit-user-select: none;
  user-select: none;
}

.lifted .grip {
  cursor: grabbing;
}

.waiting {
  cursor: progress;
}

/* Six dots: square caps on strokes of no length. */
svg {
  display: block;
  fill: none;
  stroke: currentColor;
  stroke-linecap: square;
  stroke-width: 1.5;
}
</style>
