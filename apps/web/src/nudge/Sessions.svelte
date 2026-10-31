<script lang="ts">
import Section from '../shell/Section.svelte';
import { searchedLabel } from '../shell/snapshot';
import type { Session } from './nudge';

let { sessions }: { sessions: Session[] } = $props();

// A session that hasn't reported has the zero time.
const reported = (session: Session) => new Date(session.noted_at).getFullYear() > 1;
</script>

<Section title="Agents" accent="at work">
  <div class="card">
    {#if sessions.length > 0}
      <ul class="list">
        {#each sessions as session (session.id)}
          <li>
            <div class="list-row">
              <span class="list-label">{session.project}</span>
              <span class="list-text">
                <span class="list-title">{session.note || 'No report yet'}</span>
              </span>
              <span class="list-end">
                {#if reported(session)}
                  <time datetime={session.noted_at}>{searchedLabel(session.noted_at)}</time>
                {/if}
              </span>
            </div>
          </li>
        {/each}
      </ul>
    {:else}
      <p class="empty">No agent has reported.</p>
    {/if}
  </div>
</Section>

<style>
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
