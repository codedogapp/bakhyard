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
    <p class="empty">No agent is waiting for you.</p>
  {/if}
</Section>

<style>
.questions {
  display: grid;
  gap: 24px;
}

.empty {
  margin: 0;
  color: var(--ink-muted);
}
</style>
