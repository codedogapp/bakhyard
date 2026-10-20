<script lang="ts">
let { title, accent, note }: { title: string; accent: string; note?: string } = $props();
</script>

<div class="section-head">
  <!-- The space sits inside the braces, so Svelte adds no `?? ''` fallback that no test can reach. -->
  <h2>{`${title} `}<em>{accent}</em></h2>
  {#if note}
    <p class="note">{note}</p>
  {/if}
</div>

<style>
.section-head {
  display: flex;
  flex-wrap: wrap;
  align-items: end;
  justify-content: space-between;
  gap: 6px 20px;
  margin-bottom: 30px;
  padding-top: 13px;
  border-top: 1px solid var(--ink);
}

h2 {
  margin: 0;
  font-size: 28px;
  font-weight: 500;
  letter-spacing: -0.03em;
  line-height: 1.2;
}

em {
  font-family: var(--font-serif);
  font-weight: 400;
  letter-spacing: 0;
}

.note {
  margin: 0;
  color: var(--ink-muted);
  font: 400 11px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
}

@media (max-width: 700px) {
  .section-head {
    margin-bottom: 24px;
  }

  h2 {
    font-size: 24px;
  }
}
</style>
