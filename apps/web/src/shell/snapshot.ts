/** What a poller's snapshot says of its polls, beside what they found (decisions 122 and 145). */
export interface Polls {
  /** When the latest poll to succeed began, in UTC; none before one has. */
  searched?: string;
  /** Why the latest poll failed, if it did. */
  failure?: string;
}

/** A poller's snapshot as a page has it: loading, off when its source isn't set up, failed to load, or loaded. */
export type Polled<S extends Polls> = { kind: 'loading' | 'off' | 'failed' } | { kind: 'loaded'; snapshot: S };

/**
 * Loads a poller's snapshot from url, where a 404 says that its source isn't set up; never fails, since a failed load
 * is one of the outcomes (decision 147).
 */
export async function loadSnapshot<S extends Polls>(url: string): Promise<Polled<S>> {
  try {
    const response = await fetch(url);
    if (response.status === 404) {
      return { kind: 'off' };
    }
    if (response.status !== 200) {
      return { kind: 'failed' };
    }
    const snapshot: S = await response.json();
    // A poll that succeeds sets searched, beside what it found, and one that fails sets failure: only a request that
    // waited while the service stopped gets neither.
    if (snapshot.searched === undefined && snapshot.failure === undefined) {
      return { kind: 'failed' };
    }
    return { kind: 'loaded', snapshot };
  } catch {
    return { kind: 'failed' };
  }
}

/**
 * When a search began, in the device's time zone: "26 October at 09:41". The date and the time have a format each, as
 * the Home's and the clock's, since the browsers join `dateStyle` and `timeStyle` differently.
 */
export function searchedLabel(searched: string): string {
  const date = new Date(searched);
  const day = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'long' }).format(date);
  const time = new Intl.DateTimeFormat('en-GB', { hour: '2-digit', minute: '2-digit' }).format(date);
  return `${day} at ${time}`;
}
