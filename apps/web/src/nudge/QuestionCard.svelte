<script lang="ts">
import { searchedLabel } from '../shell/snapshot';
import { type Question, sendAnswer } from './nudge';

let { question, answered }: { question: Question; answered: () => void } = $props();

let text = $state('');
let sending = $state(false);
let failure = $state('');

// An option's tap sends it at once, with what the user typed, if anything; the reply alone needs its button.
async function send(option?: string) {
  const reply = text.trim();
  if (sending || (!option && !reply)) {
    return;
  }
  sending = true;
  failure = (await sendAnswer(question.id, { option, text: reply || undefined })) ?? '';
  sending = false;
  if (!failure) {
    answered();
  }
}
</script>

<article class="question">
  <p class="meta">
    <span>{question.project}</span>
    <time datetime={question.asked_at}>{searchedLabel(question.asked_at)}</time>
  </p>
  <h3>{question.text}</h3>
  {#if question.details}
    <p class="details">{question.details}</p>
  {/if}
  {#if question.options.length > 0}
    <div class="options">
      {#each question.options as option (option)}
        <button type="button" class="form-button" aria-disabled={sending} onclick={() => send(option)}>
          {option}
        </button>
      {/each}
    </div>
  {/if}
  <form
    class="form-row"
    onsubmit={(event) => {
      event.preventDefault();
      void send();
    }}
  >
    <input
      class="form-field"
      type="text"
      aria-label={`Reply to: ${question.text}`}
      placeholder="Type a reply"
      bind:value={text}
    >
    <button type="submit" class="form-button" aria-disabled={sending}>Send</button>
    <p class="form-status" role="status">{failure}</p>
  </form>
</article>

<style>
.question {
  padding: 20px;
  border: 1px solid var(--rule);
  background: var(--surface);
  box-shadow: var(--offset) var(--offset) 0 var(--shadow);
}

.meta {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin: 0 0 8px;
  color: var(--ink-muted);
  font: 400 11px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
  overflow-wrap: anywhere;
}

h3 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.details {
  margin: 8px 0 0;
  color: var(--ink-muted);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.options {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin: 20px 0 12px;
}

.form-row {
  margin-top: 16px;
}

.options + .form-row {
  margin-top: 0;
}
</style>
