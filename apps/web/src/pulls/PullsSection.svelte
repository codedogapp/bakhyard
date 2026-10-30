<script lang="ts">
import PolledSection from '../shell/PolledSection.svelte';
import { type Pulls, reviewLine } from './pulls';

// One of the PRs page's lists, in a section under the head; the texts say when it's empty and when it couldn't load.
// With showReviewers, each row also says where its PR stands with its reviewers (decision 174).
let {
  pulls,
  title,
  accent,
  empty,
  failed,
  showReviewers = false,
}: { pulls: Pulls; title: string; accent: string; empty: string; failed: string; showReviewers?: boolean } = $props();
</script>

<PolledSection polled={pulls} list={(snapshot) => snapshot.pulls} {title} {accent} noun="PR" {empty} {failed}>
  {#snippet row(pull)}
    <!-- A PR without a link to Bitbucket gets a row that isn't a link. -->
    <svelte:element this={pull.url ? 'a' : 'div'} class="list-row" href={pull.url}>
      <span class="list-label">#{pull.id}</span>
      <span class="list-text">
        <span class="list-title">{pull.title}</span>
        <span class="list-detail">{pull.repo}</span>
        {#if showReviewers}
          <span class="list-detail">{reviewLine(pull.reviewers)}</span>
        {/if}
      </span>
      <span class="list-end">
        {#if pull.draft}
          <span class="chip">Draft</span>
        {/if}
      </span>
    </svelte:element>
  {/snippet}
</PolledSection>
