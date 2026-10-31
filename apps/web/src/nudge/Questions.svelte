<script lang="ts">
import Section from '../shell/Section.svelte';
import type { Question } from './nudge';
import QuestionCard from './QuestionCard.svelte';

let { questions, answered }: { questions: Question[]; answered: () => void } = $props();
const note = $derived(questions.length === 0 ? undefined : `${questions.length} waiting`);
</script>

<Section title="Waiting" accent="for you" {note}>
  {#if questions.length > 0}
    <div class="questions">
      {#each questions as question (question.id)}
        <QuestionCard {question} {answered} />
      {/each}
    </div>
  {:else}
    <div class="card">
      <p class="empty">No agent is waiting for you.</p>
    </div>
  {/if}
</Section>

<style>
.questions {
  display: grid;
  gap: 24px;
}

.card {
  border: 1px solid var(--rule);
  background: var(--surface);
  box-shadow: var(--offset) var(--offset) 0 var(--shadow);
}

.empty {
  margin: 0;
  padding: 20px;
  color: var(--ink-muted);
}
</style>
