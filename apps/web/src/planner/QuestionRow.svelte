<script lang="ts">
import { AnswerState } from './answer.svelte';
import type { Question } from './conversation';
import { authorLabel, sendLabel } from './labels';

// A question of the agent's that waits, of the plan with the ID as its page's path holds it, and the user's answer.
let { id, question }: { id: string; question: Question } = $props();

const answer = new AnswerState();

function submit(event: SubmitEvent): void {
  event.preventDefault();
  void answer.send(id, question.id);
}
</script>

<li class="list-row question">
  <span class="list-label">{authorLabel('assistant')}</span>
  <div class="list-text">
    <p class="text">{question.text}</p>
    {#if question.choices.length > 0}
      <div class="choices">
        <!-- Unkeyed: nothing keeps the agent from offering the same choice twice. -->
        {#each question.choices as choice}
          <button
            class="form-button"
            type="button"
            aria-disabled={answer.busy}
            onclick={() => answer.send(id, question.id, choice)}
          >
            {choice}
          </button>
        {/each}
      </div>
    {/if}
    {#if question.freeform}
      <!-- No maxlength: the service limits the answer (decision 411), and a refused one stays in the field to edit. -->
      <form class="form-row" onsubmit={submit}>
        <input
          class="form-field"
          type="text"
          aria-label={`Answer to: ${question.text}`}
          placeholder="Type an answer"
          readonly={answer.busy}
          bind:value={answer.text}
        >
        <!-- aria-disabled, not disabled, so focus stays on the button while it waits. -->
        <button class="form-button" type="submit" aria-disabled={answer.busy}>Send</button>
      </form>
    {/if}
    <p class="form-status" role="status">{sendLabel(answer.status)}</p>
  </div>
</li>

<style>
/* The label stays by the question's first line, as a message's does by its own. */
.question {
  align-content: center;
  align-items: baseline;
}

/* The question keeps its line breaks, as a message does. */
.text {
  margin: 0;
  white-space: pre-wrap;
}

.choices {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 12px;
}

/* A long choice wraps rather than widen the page. */
.choices .form-button {
  max-width: 100%;
}

.form-row {
  margin-top: 12px;
}
</style>
