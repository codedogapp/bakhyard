<script lang="ts">
import PageHead from '../shell/PageHead.svelte';
import Section from '../shell/Section.svelte';
import { type Chat, followPlan } from './follow';
import { authorLabel, workLabel } from './labels';
import QuestionRow from './QuestionRow.svelte';
import ReplyForm from './ReplyForm.svelte';

// The plan's conversation, followed on its stream while the page shows.
let { id }: { id: string } = $props();

let chat = $state.raw<Chat>({ kind: 'loading' });

// Another plan's page starts from nothing, rather than show this plan's conversation until its own comes.
$effect(() => {
  chat = { kind: 'loading' };
  return followPlan(id, (next) => {
    chat = next;
  });
});
</script>

<PageHead>Plan</PageHead>
{#if chat.kind === 'failed'}
  <p class="page-status">Couldn't load the plan.</p>
{:else if chat.kind === 'loaded'}
  {@const messages = [...chat.talk.messages, ...chat.talk.writing]}
  {@const questions = chat.talk.questions}
  <Section title="The" accent="conversation">
    <div class="card">
      {#if messages.length > 0 || questions.length > 0}
        <ul class="list">
          <!-- Unkeyed: rows come and go only at the end, so a message that comes whole keeps its row. -->
          {#each messages as message}
            <li class="list-row message">
              <span class="list-label">{authorLabel(message.from)}</span>
              <span class="list-text text">{message.text}</span>
            </li>
          {/each}
          <!-- Keyed, so each answer stays with its question as another goes. -->
          {#each questions as question (question.id)}
            <QuestionRow {id} {question} />
          {/each}
        </ul>
      {:else}
        <p class="empty">No messages yet.</p>
      {/if}
    </div>
    <p class="form-status work" role="status">{workLabel(chat.talk)}</p>
    <!-- Copilot takes no message while a question waits (decision 414), so the reply hides, keeping its text. -->
    <div hidden={questions.length > 0}>
      <ReplyForm {id} />
    </div>
  </Section>
{/if}

<style>
.card {
  border: 1px solid var(--rule);
  background: var(--surface);
  box-shadow: var(--offset) var(--offset) 0 var(--shadow);
}

/* A long message's label stays by its first line, and a short one's row is centred as the other lists' rows are. */
.message {
  align-content: center;
  align-items: baseline;
}

/* A message keeps its line breaks. */
.text {
  white-space: pre-wrap;
}

/* The CLI's error can hold a long word, such as a link, which breaks rather than widen the page. */
.work {
  overflow-wrap: anywhere;
}

.empty {
  margin: 0;
  padding: 20px;
  color: var(--ink-muted);
}
</style>
