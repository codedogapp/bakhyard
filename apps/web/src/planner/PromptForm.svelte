<script lang="ts">
import { startLabel } from './labels';
import { PromptState } from './prompt.svelte';

// started tells the page that a plan has started, so that it lists it.
let { started }: { started: () => void } = $props();

const prompt = new PromptState();

function submit(event: SubmitEvent): void {
  event.preventDefault();
  void prompt.start(started);
}
</script>

<!-- No maxlength: the service limits the prompt (decision 405), and a refused one stays in the field to edit. -->
<form class="form-row" onsubmit={submit}>
  <textarea
    class="form-field"
    aria-label="Prompt"
    placeholder="What should Copilot plan?"
    rows="3"
    readonly={prompt.busy}
    bind:value={prompt.text}
  ></textarea>
  <!-- aria-disabled, not disabled, so focus stays on the button while it waits. -->
  <button class="form-button" type="submit" aria-disabled={prompt.busy}>Start</button>
  <p class="form-status" role="status">{startLabel(prompt.status)}</p>
</form>

<style>
/* Over the plans, the form leaves the space that the Todos page's form leaves over its first section. */
form {
  margin: 0 0 30px;
  padding: 0 var(--page-padding);
}

/* The prompt takes the row, with Start under it. */
textarea {
  flex-basis: 100%;
  margin-bottom: 12px;
  resize: vertical;
}
</style>
