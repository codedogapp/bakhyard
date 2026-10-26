<script lang="ts">
import { addLabel } from './labels';
import type { TodosState } from './state.svelte';

let { list, field = $bindable() }: { list: TodosState; field?: HTMLInputElement } = $props();

function submit(event: SubmitEvent): void {
  event.preventDefault();
  void list.add();
}
</script>

<!-- No maxlength: the service checks the title (decision 106), and a refused one stays in the field to edit. -->
<form class="form-row" onsubmit={submit}>
  <input
    class="form-field"
    type="text"
    aria-label="New todo"
    placeholder="New todo"
    readonly={list.busy}
    bind:value={list.draft}
    bind:this={field}
  >
  <!-- aria-disabled, not disabled, so focus stays on the button while it waits. -->
  <button class="form-button" type="submit" aria-disabled={list.busy}>Add</button>
  <p class="form-status" role="status">{addLabel(list.addStatus)}</p>
</form>

<style>
form {
  margin-bottom: 30px;
}

@media (max-width: 700px) {
  form {
    margin-bottom: 24px;
  }
}
</style>
