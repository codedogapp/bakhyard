import { fetchHome, type Home } from './api';

const POLL_MS = 60_000;

/** Whether the last load reached the Home; "connecting" until the first one ends. */
export type Connection = 'connecting' | 'online' | 'offline';

/** What the app knows of the Home: the last one loaded, when it loaded, and whether the last load worked. */
export interface Feed {
  connection: Connection;
  home?: Home;
  asOf?: Date;
}

/**
 * Loads the Home at once, every minute while the page is visible and whenever it becomes visible again,
 * and reports each result until the returned function is called. Only the latest load counts, and a failed
 * one keeps the last Home.
 */
export function pollHome(onChange: (feed: Feed) => void): () => void {
  let feed: Feed = { connection: 'connecting' };
  let latest = 0;
  let stopped = false;

  const load = async () => {
    const request = ++latest;
    const home = await fetchHome().catch(() => undefined);
    if (stopped || request !== latest) {
      return;
    }
    feed = home ? { connection: 'online', home, asOf: new Date() } : { ...feed, connection: 'offline' };
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
