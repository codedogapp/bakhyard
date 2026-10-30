<script lang="ts">
import { onMount } from 'svelte';

import { countLabel } from '../shell/count';
import { dayLabel } from '../shell/day';
import PageHead from '../shell/PageHead.svelte';
import Section from '../shell/Section.svelte';
import { loadPlans, type Plans } from './api';
import { titleLabel } from './labels';
import PromptForm from './PromptForm.svelte';

// The plans, loaded each time the page shows and after each start; now tells which day is today.
let { now }: { now: Date } = $props();

let list = $state.raw<Plans>({ kind: 'loading' });

// The list stays as it is until the new one comes.
function load(): void {
  void loadPlans().then((loaded) => {
    list = loaded;
  });
}

onMount(load);
</script>

<PageHead summary="Plans that Copilot works out with you.">Planner</PageHead>
{#if list.kind === 'off'}
  <p class="page-status">Copilot isn't set up.</p>
{:else if list.kind !== 'loading'}
  <PromptForm started={load} />
{/if}
{#if list.kind === 'failed'}
  <Section title="Your" accent="plans">
    <p class="muted">Couldn't load the plans.</p>
  </Section>
{:else if list.kind === 'loaded'}
  <Section title="Your" accent="plans" note={countLabel(list.plans.length, 'plan')}>
    <div class="card">
      {#if list.plans.length > 0}
        <ul class="list">
          {#each list.plans as plan (plan.id)}
            <li>
              <div class="list-row">
                <time class="list-label" datetime={plan.updated_at}>{dayLabel(plan.updated_at, now)}</time>
                <span class="list-text">
                  <span class="list-title">{titleLabel(plan.title)}</span>
                </span>
              </div>
            </li>
          {/each}
        </ul>
      {:else}
        <p class="empty">No plans yet.</p>
      {/if}
    </div>
  </Section>
{/if}

<style>
.muted {
  margin: 0;
  color: var(--ink-muted);
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
