<script lang="ts">
import { isCurrent, navItems } from './nav';

let { path }: { path: string } = $props();
</script>

<header class="header">
  <a class="wordmark" href="/">bakhyard</a>
  <nav class="nav" aria-label="Apps">
    <ul>
      {#each navItems as item (item.href)}
        <li>
          <a href={item.href} aria-current={isCurrent(item.href, path) ? 'page' : undefined}>{item.label}</a>
        </li>
      {/each}
    </ul>
  </nav>
  <!-- The gateway ends this device's session and sends the browser to /, where the app shows its sign-in screen. -->
  <form class="sign-out" method="post" action="/auth/logout">
    <button type="submit">Sign out</button>
  </form>
</header>

<style>
.header {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: 32px;
  align-items: center;
  padding: 24px var(--page-padding);
  border-bottom: 1px solid var(--rule);
}

.wordmark {
  justify-self: start;
  color: var(--ink);
  font-size: 25px;
  font-weight: 600;
  letter-spacing: -0.07em;
  line-height: 1;
  text-decoration: none;
}

ul {
  display: flex;
  gap: 32px;
  margin: 0;
  padding: 0;
  list-style: none;
}

/* It stays at the right end where the nav gives way to the tab bar. */
.sign-out {
  grid-column: 3;
  justify-self: end;
  margin: 0;
}

.sign-out button {
  /* The tap target keeps the padding, but the header's row makes room only for the text. */
  margin-block: -4px;
  border: 0;
  background: none;
  cursor: pointer;
}

/* Sign out looks like the nav's links. */
.nav a,
.sign-out button {
  position: relative;
  display: block;
  padding: 4px 0;
  color: var(--ink);
  font: 400 11.5px / 1.6 var(--font-mono);
  letter-spacing: 0.035em;
  text-decoration: none;
  text-transform: uppercase;
}

.nav a::after,
.sign-out button::after {
  content: "";
  position: absolute;
  right: 0;
  bottom: -2px;
  left: 0;
  height: 1px;
  background: var(--ink);
  transform: scaleX(0);
  transform-origin: left;
}

.nav a:hover::after,
.nav a[aria-current="page"]::after,
.sign-out button:hover::after {
  transform: scaleX(1);
}

@media (prefers-reduced-motion: no-preference) {
  .nav a::after,
  .sign-out button::after {
    transition: transform 250ms ease;
  }
}

/* The bottom tab bar takes over the nav here (plan §7). */
@media (max-width: 980px) {
  .header {
    padding-block: 18px;
  }

  .nav {
    display: none;
  }
}

@media (max-width: 700px) {
  .header {
    padding-block: 16px;
  }

  .wordmark {
    font-size: 17px;
  }
}
</style>
