<script lang="ts">
import { onMount } from 'svelte';

import PageHead from '../shell/PageHead.svelte';
import Section from '../shell/Section.svelte';
import AddForm from './AddForm.svelte';
import DeleteDialog from './DeleteDialog.svelte';
import { Drag } from './drag.svelte';
import { Editor } from './editor.svelte';
import { Focus } from './focus';
import { Inbox } from './inbox.svelte';
import JiraInbox from './JiraInbox.svelte';
import JiraTickets from './JiraTickets.svelte';
import { groupNote, priorityLabel, todosSummary } from './labels';
import { TodosState } from './state.svelte';
import TodoRow from './TodoRow.svelte';
import { doneTickets, loadTickets, openTickets, type Tickets } from './tickets';

const list = new TodosState();
let addField = $state<HTMLInputElement>();
const focus = new Focus(() => addField?.focus());
const editor = new Editor(list, focus);
const drag = new Drag(list, editor);
let tickets = $state.raw<Tickets>({ kind: 'loading' });
const open = $derived(openTickets(tickets));
const done = $derived(doneTickets(tickets));
const inbox = new Inbox(list, () => tickets, focus);
// The todos once the tickets and the dismissed tickets have loaded too, so the inbox above them never pushes them down.
const todos = $derived(tickets.kind === 'loading' || inbox.loading ? undefined : list.todos);

onMount(() => {
  void list.load();
  void inbox.load();
  void loadTickets().then((loaded) => {
    tickets = loaded;
  });
});
</script>

<PageHead summary={todos && todosSummary(todos.length)}>Todos</PageHead>
{#if todos}
  <div class={['tools', { last: list.groups.length === 0 && inbox.tickets.length === 0 }]}>
    <AddForm {list} bind:field={addField} />
  </div>
  <JiraInbox {inbox} />
  {#each list.groups as group (group.priority)}
    <Section title={priorityLabel(group.priority)} accent="priority" note={groupNote(group.todos.length)}>
      <ul class="card">
        {#each group.todos as todo (todo.id)}
          <TodoRow {list} {todo} {editor} {focus} {drag} {open} {done} />
        {/each}
      </ul>
    </Section>
  {/each}
{:else if list.failed}
  <p class="failed">Couldn't load the todos.</p>
{/if}
<!-- The tickets wait for the todos to show or fail, so the todos never push them down. -->
{#if todos || list.failed}
  <JiraTickets {tickets} />
{/if}
<DeleteDialog {editor} />

<style>
.tools {
  padding: 0 var(--page-padding);
}

/* Without a section under it, the form ends the page, with the space a section leaves under it. */
.last {
  padding-bottom: 64px;
}

.card {
  margin: 0;
  padding: 0;
  border: 1px solid var(--rule);
  background: var(--surface);
  box-shadow: var(--offset) var(--offset) 0 var(--shadow);
  list-style: none;
}

.failed {
  margin: 0;
  padding: 0 var(--page-padding) 64px;
  color: var(--ink-muted);
}

@media (max-width: 700px) {
  .last,
  .failed {
    padding-bottom: 48px;
  }
}
</style>
