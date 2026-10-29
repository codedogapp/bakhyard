<script lang="ts">
import { onMount } from 'svelte';

import PageHead from '../shell/PageHead.svelte';
import PullsSection from './PullsSection.svelte';
import { loadPulls, loadReviews, type Pulls } from './pulls';

let mine = $state.raw<Pulls>({ kind: 'loading' });
let reviews = $state.raw<Pulls>({ kind: 'loading' });

// The lists load at once, so my PRs needn't wait for the reviews, whose first poll can take a while (decision 172).
onMount(() => {
  void loadPulls().then((loaded) => {
    mine = loaded;
  });
  void loadReviews().then((loaded) => {
    reviews = loaded;
  });
});
</script>

<PageHead>Pull requests</PageHead>
<!-- Without Bitbucket, the pulls service answers 404 for both lists; one older than the reviews, for those alone. -->
{#if mine.kind === 'off' && reviews.kind === 'off'}
  <p class="off">Bitbucket isn't set up.</p>
{:else}
  <PullsSection
    pulls={mine}
    title="Your"
    accent="PRs"
    empty="No open pull requests."
    failed="Couldn't load the pull requests."
  />
  <PullsSection
    pulls={reviews}
    title="Review"
    accent="requests"
    empty="No pull requests to review."
    failed="Couldn't load the review requests."
  />
{/if}

<style>
.off {
  margin: 0;
  padding: 0 var(--page-padding) 64px;
  color: var(--ink-muted);
}

@media (max-width: 700px) {
  .off {
    padding-bottom: 48px;
  }
}
</style>
