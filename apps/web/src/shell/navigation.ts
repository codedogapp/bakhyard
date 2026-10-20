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

// The browser keeps what it can't intercept (another origin, another document's history entry), a jump to an anchor,
// a download, a form's submission and a reload, which loads the page afresh.
function isInterceptable(event: NavigateEvent): boolean {
  return (
    event.canIntercept &&
    !event.hashChange &&
    event.downloadRequest === null &&
    event.formData === null &&
    event.navigationType !== 'reload'
  );
}
