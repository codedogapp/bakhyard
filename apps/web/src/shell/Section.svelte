<script lang="ts">
import type { Snippet } from 'svelte';

import Arrow from './Arrow.svelte';

interface Link {
  href: string;
  label: string;
}

let {
  title,
  accent,
  note,
  link,
  children,
}: { title: string; accent: string; note?: string; link?: Link; children: Snippet } = $props();
</script>

<section class="section">
  <div class="head">
    <!-- The space sits inside the braces, so Svelte adds no `?? ''` fallback that no test can reach. -->
    <h2>{`${title} `}<em>{accent}</em></h2>
    {#if note}
      <p class="note">{note}</p>
    {/if}
    {#if link}
      <a class="link" href={link.href}>{link.label}<Arrow /></a>
    {/if}
  </div>
  {@render children()}
</section>

<style>
.section {
  padding: 0 var(--page-padding) 64px;
}

.head {
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

/* The underline is a border, so it runs under the arrow too. */
.link {
  display: inline-flex;
  align-items: center;
  gap: 22px;
  padding-bottom: 5px;
  border-bottom: 1px solid var(--ink-muted);
  color: inherit;
  font-size: 14px;
  line-height: 1.5;
  text-decoration: none;
  white-space: nowrap;
}

.link:hover {
  border-color: var(--ink);
}

@media (max-width: 700px) {
  .section {
    padding-bottom: 48px;
  }

  .head {
    margin-bottom: 24px;
  }

  h2 {
    font-size: 24px;
  }
}
</style>
