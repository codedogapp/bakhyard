/**
 * Shows each page without loading a new document: intercepts the browser's own navigations and runs onPath with the
 * new path as their handler. The browser still updates the URL and, once onPath settles, resets focus and scrolls to
 * the top, or back to where the page was left on back and forward. Returns a function that stops.
 */
export function followNavigation(onPath: (path: string) => Promise<void>): () => void {
  // Without the Navigation API, each link loads its page in full, which works too.
  if (!('navigation' in globalThis)) {
    return () => undefined;
  }
  const onNavigate = (event: NavigateEvent) => {
    if (!isInterceptable(event)) {
      return;
    }
    const { pathname } = new URL(event.destination.url);
    event.intercept({ handler: () => onPath(pathname) });
  };
  const listening = new AbortController();
  navigation.addEventListener('navigate', onNavigate, { signal: listening.signal });
  return () => listening.abort();
}

/**
 * Opens the page at path as a link to it would. The browser fires a navigate event for it, so followNavigation shows
 * the page in place where the browser has the Navigation API; elsewhere it loads in full.
 */
export function openPage(path: string): void {
  location.assign(path);
}

// The gateway serves the paths under /auth/ itself: signing in starts there.
const authPrefix = '/auth/';

// The browser keeps what it can't intercept (another origin, another document's history entry), a jump to an anchor,
// a download, a form's submission, a reload, which loads the page afresh, and the paths under /auth/.
function isInterceptable(event: NavigateEvent): boolean {
  return (
    event.canIntercept &&
    !event.hashChange &&
    event.downloadRequest === null &&
    event.formData === null &&
    event.navigationType !== 'reload' &&
    !new URL(event.destination.url).pathname.startsWith(authPrefix)
  );
}
