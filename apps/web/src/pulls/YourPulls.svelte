<script lang="ts">
import Section from '../shell/Section.svelte';
import { searchedLabel } from '../shell/snapshot';
import { type Pulls, pullsNote } from './pulls';

let { pulls }: { pulls: Pulls } = $props();
</script>

{#if pulls.kind === 'loaded'}
  {@const { pulls: list, searched, failure } = pulls.snapshot}
  <Section title="Your" accent="PRs" note={list && pullsNote(list.length)}>
    {#if failure}
      <p class="muted">
        The last search failed: {failure}.
        {#if list && searched}
          These PRs are from <time datetime={searched}>{searchedLabel(searched)}</time>.
        {/if}
      </p>
    {/if}
    {#if list}
      <div class="card">
        {#if list.length > 0}
          <ul class="list">
            {#each list as pull}
              <li>
                <!-- A PR without a link to Bitbucket gets a row that isn't a link. -->
                <svelte:element this={pull.url ? 'a' : 'div'} class="list-row" href={pull.url}>
                  <span class="list-label">#{pull.id}</span>
                  <span class="list-text">
                    <span class="list-title">{pull.title}</span>
                    <span class="list-detail">{pull.repo}</span>
                  </span>
                  <span class="list-end">
                    {#if pull.draft}
                      <span class="chip">Draft</span>
                    {/if}
                  </span>
                </svelte:element>
              </li>
            {/each}
          </ul>
        {:else}
          <p class="empty">No open pull requests.</p>
        {/if}
      </div>
    {/if}
  </Section>
{:else if pulls.kind === 'failed'}
  <Section title="Your" accent="PRs">
    <p class="muted">Couldn't load the pull requests.</p>
  </Section>
{/if}

<style>
.muted {
  margin: 0;
  color: var(--ink-muted);
  overflow-wrap: anywhere;
}

.card {
  border: 1px solid var(--rule);
  background: var(--surface);
  box-shadow: var(--offset) var(--offset) 0 var(--shadow);
}

.muted + .card {
  margin-top: 20px;
}

.empty {
  margin: 0;
  padding: 20px;
  color: var(--ink-muted);
}
</style>
