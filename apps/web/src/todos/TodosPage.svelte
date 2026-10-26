<script lang="ts">
import { onMount } from 'svelte';

import PageHead from '../shell/PageHead.svelte';
import AddForm from './AddForm.svelte';
import { todosSummary } from './labels';
import { TodosState } from './state.svelte';
import TodoRow from './TodoRow.svelte';

const list = new TodosState();

onMount(() => {
  void list.load();
});
</script>

<PageHead summary={list.todos && todosSummary(list.todos.length)}>Todos</PageHead>
<div class="todos">
  {#if list.todos}
    <AddForm {list} />
    {#if list.todos.length > 0}
      <ul class="card">
        {#each list.todos as todo (todo.id)}
          <TodoRow {list} {todo} />
        {/each}
      </ul>
    {/if}
  {:else if list.failed}
    <p class="failed">Couldn't load the todos.</p>
  {/if}
</div>

<style>
.todos {
  padding: 0 var(--page-padding) 64px;
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
  color: var(--ink-muted);
}

@media (max-width: 700px) {
  .todos {
    padding-bottom: 48px;
  }
}
</style>
