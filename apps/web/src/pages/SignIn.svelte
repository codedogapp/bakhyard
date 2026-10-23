<script lang="ts">
import PageHead from '../shell/PageHead.svelte';
</script>

<PageHead summary="Sign in with your Google account to see your day.">Signed out</PageHead>
<div class="actions">
  <!-- The gateway serves this path: the browser loads it in full, and Google's sign-in follows. -->
  <a class="button" href="/auth/google/login">Sign in</a>
</div>

<style>
.actions {
  padding: 0 var(--page-padding) 50px;
}

.button {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  padding: 14px 22px;
  border: 1px solid var(--ink);
  background: var(--ink);
  color: var(--paper);
  font: 400 12px / 1.2 var(--font-mono);
  letter-spacing: 0.045em;
  text-decoration: none;
  text-transform: uppercase;
}

.button:hover {
  background: var(--paper);
  color: var(--ink);
}
</style>
