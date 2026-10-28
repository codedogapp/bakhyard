/** How often a page that is on screen loads again, in case the live stream and the pushes both miss a change. */
export const fallbackMs = 60_000;

/**
 * Calls refresh whenever the page may be out of date: when the service says on its event stream that something
 * changed, when the service worker says that a push came in, when the page comes back on screen, and every fallbackMs
 * while it is on screen. Returns what stops it.
 */
export function stayCurrent(refresh: () => void): () => void {
  const events = new EventSource('/api/nudge/events');
  events.addEventListener('changed', refresh);
  // The stream opens again by itself after a break, and a change during the break is read by this. The first open
  // follows the page's own first load, so it needs no other.
  let opened = false;
  events.addEventListener('open', () => {
    if (opened) {
      refresh();
    }
    opened = true;
  });

  const onVisible = () => {
    if (document.visibilityState === 'visible') {
      refresh();
    }
  };
  document.addEventListener('visibilitychange', onVisible);

  // The worker tells every open page about a push, whether or not the stream is up.
  const worker = navigator.serviceWorker;
  worker?.addEventListener('message', refresh);

  const timer = setInterval(onVisible, fallbackMs);

  return () => {
    events.close();
    document.removeEventListener('visibilitychange', onVisible);
    worker?.removeEventListener('message', refresh);
    clearInterval(timer);
  };
}
