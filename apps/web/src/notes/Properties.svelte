<script lang="ts">
import type { Property } from './properties';

// A note's Properties block: each key, then its value as text or a link, or its list's values as chips (plan §7).
let { properties }: { properties: readonly Property[] } = $props();
</script>

{#if properties.length > 0}
  <dl>
    {#each properties as property}
      <div>
        <dt>{property.key}</dt>
        <dd>
          {#if Array.isArray(property.value)}
            <ul>
              {#each property.value as value}
                <li>{value}</li>
              {/each}
            </ul>
          {:else if typeof property.value === 'string'}
            {property.value}
          {:else if property.value.href === undefined}
            <span class="unresolved">{property.value.text}</span>
          {:else}
            <a href={property.value.href}>{property.value.text}</a>
          {/if}
        </dd>
      </div>
    {/each}
  </dl>
{/if}

<style>
dl {
  display: grid;
  gap: 10px;
  margin: 0 0 32px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--rule);
}

div {
  display: grid;
  grid-template-columns: 120px minmax(0, 1fr);
  gap: 16px;
  align-items: baseline;
}

dt {
  color: var(--ink-muted);
  font: 400 11px / 1.6 var(--font-mono);
  letter-spacing: 0.045em;
  overflow-wrap: anywhere;
  text-transform: uppercase;
}

dd {
  margin: 0;
  overflow-wrap: anywhere;
}

/* As the note's links, and its wikilinks that lead nowhere, show. */
a {
  color: var(--ink);
  text-underline-offset: 2px;
}

.unresolved {
  color: var(--ink-muted);
}

ul {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 0;
  padding: 0;
  list-style: none;
}

li {
  padding: 3px 6px;
  border: 1px solid var(--ink);
  font: 400 10px / 1.2 var(--font-mono);
  letter-spacing: 0.045em;
  text-transform: uppercase;
}
</style>
