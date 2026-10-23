import { fetchHome, type Home, SignedOutError } from './api';

const POLL_MS = 60_000;

/** Whether the last load reached the Home; "connecting" until the first one ends, and "signed-out" when the user has no session. */
export type Connection = 'connecting' | 'online' | 'offline' | 'signed-out';

/** What the app knows of the Home: the last one loaded, when it loaded, and whether the last load worked. Signed out, it knows none. */
export interface Feed {
  connection: Connection;
  home?: Home;
  asOf?: Date;
}

/**
 * Loads the Home at once, every minute while the page is visible and whenever it becomes visible again,
 * and reports each result until the returned function is called. Only the latest load counts, and a failed
 * one keeps the last Home, unless the user has no session.
 */
export function pollHome(onChange: (feed: Feed) => void): () => void {
  let feed: Feed = { connection: 'connecting' };
  let latest = 0;
  let stopped = false;

  const load = async () => {
    const request = ++latest;
    let next: Feed;
    try {
      next = { connection: 'online', home: await fetchHome(), asOf: new Date() };
    } catch (error) {
      next = error instanceof SignedOutError ? { connection: 'signed-out' } : { ...feed, connection: 'offline' };
    }
    if (stopped || request !== latest) {
      return;
    }
    feed = next;
    onChange(feed);
  };

  const loadIfVisible = () => {
    if (document.visibilityState === 'visible') {
      void load();
    }
  };

  void load();
  const timer = setInterval(loadIfVisible, POLL_MS);
  document.addEventListener('visibilitychange', loadIfVisible);

  return () => {
    stopped = true;
    clearInterval(timer);
    document.removeEventListener('visibilitychange', loadIfVisible);
  };
}
