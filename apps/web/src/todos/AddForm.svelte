<script lang="ts">
import { addLabel } from './labels';
import type { TodosState } from './state.svelte';

let { list }: { list: TodosState } = $props();

function submit(event: SubmitEvent): void {
  event.preventDefault();
  void list.add();
}
</script>

<!-- No maxlength: the service checks the title (decision 106), and a refused one stays in the field to edit. -->
<form onsubmit={submit}>
  <input type="text" aria-label="New todo" placeholder="New todo" readonly={list.busy} bind:value={list.draft}>
  <!-- aria-disabled, not disabled, so focus stays on the button while it waits. -->
  <button type="submit" aria-disabled={list.busy}>Add</button>
  <p class="status" role="status">{addLabel(list.addStatus)}</p>
</form>

<style>
/* The add's status takes a line of its own under the field, once it has something to say. */
form {
  display: flex;
  flex-wrap: wrap;
  gap: 0 12px;
  margin-bottom: 30px;
}

input {
  flex: 1 1 0;
  min-width: 0;
  min-height: 44px;
  padding: 10px 14px;
  border: 1px solid var(--rule);
  border-radius: 0;
  background: var(--surface);
  color: inherit;
  /* 16 px, so iOS doesn't zoom in on focus. */
  font: inherit;
  appearance: none;
}

input:hover {
  border-color: var(--ink);
}

input::placeholder {
  color: var(--ink-muted);
}

button {
  flex: none;
  min-height: 44px;
  padding: 14px 22px;
  border: 1px solid var(--ink);
  border-radius: 0;
  background: var(--surface);
  color: var(--ink);
  cursor: pointer;
  font: 400 12px / 1.2 var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
}

button:hover {
  background: var(--fill);
}

button[aria-disabled="true"] {
  color: var(--ink-muted);
  cursor: default;
}

/* Never hidden: a live region has to be there before its first change for screen readers to announce it. */
.status {
  flex-basis: 100%;
  margin: 0;
  color: var(--ink-muted);
  font: 400 11px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
}

.status:not(:empty) {
  margin-top: 12px;
}

@media (max-width: 700px) {
  form {
    margin-bottom: 24px;
  }
}
</style>
