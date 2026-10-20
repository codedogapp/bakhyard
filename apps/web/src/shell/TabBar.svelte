<script lang="ts">
import { tabCurrent, tabItems } from './nav';

let { path }: { path: string } = $props();
</script>

<nav class="tab-bar" aria-label="Apps">
  <ul>
    {#each tabItems as item (item.href)}
      <li>
        <a href={item.href} aria-current={tabCurrent(item.href, path)}>{item.label}</a>
      </li>
    {/each}
  </ul>
</nav>

<style>
.tab-bar {
  display: none;
}

/* It takes over from the header's nav (plan §7). */
@media (max-width: 980px) {
  /* The bar's 49 px are its 48 px cells and its border. The page ends above it, so the bar never covers it. */
  :global(body) {
    padding-bottom: calc(49px + env(safe-area-inset-bottom));
  }

  .tab-bar {
    position: fixed;
    right: 0;
    bottom: 0;
    left: 0;
    display: block;
    padding: 0 env(safe-area-inset-right) env(safe-area-inset-bottom) env(safe-area-inset-left);
    border-top: 1px solid var(--ink);
    background: var(--surface);
  }
}

ul {
  display: grid;
  grid-auto-columns: 1fr;
  grid-auto-flow: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

li + li {
  border-left: 1px solid var(--rule);
}

a {
  display: grid;
  place-items: center;
  height: 48px;
  color: var(--ink);
  font: 400 10px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
  text-decoration: none;
  text-transform: uppercase;
}

a:hover {
  background: var(--fill);
}

a[aria-current] {
  background: var(--ink);
  color: var(--paper);
}

/* The bar touches the screen's edges, so the ring goes inside the cell, in the cell's text colour. */
a:focus-visible {
  outline-color: currentColor;
  outline-offset: -5px;
}
</style>
