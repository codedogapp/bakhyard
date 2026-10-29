import type { Week } from '../time/week';

/** The apps that have items so far, as home names them. */
type App = 'github' | 'planner' | 'pulls' | 'time' | 'todos';

/** Where the items come from so far: an outside service, or Bakhyard itself. */
type Source = 'Bakhyard' | 'Bitbucket' | 'Copilot' | 'GitHub' | 'Jira';

/** An item that needs the user, with only the fields the app reads so far. */
export interface Item {
  id: string;
  app: App;
  title: string;
  detail?: string;
  source: Source;
  /** When it's due, in RFC 3339; absent when it has no due time. */
  due?: string;
  late: boolean;
}

/** How home's asking went: the peer answered, or it failed or didn't answer in time. */
type State = 'ok' | 'unavailable';

/** How asking one of home's sources went, with only the field the app reads so far. */
interface SourceStatus {
  state: State;
}

/** The Home screen's data from GET /api/home, with only the fields the app reads so far. */
export interface Home {
  name: string;
  items: Item[];
  /** One per source home asks; absent from the answer of a home older than the field. */
  sources?: SourceStatus[];
  /** Absent when home has no week: the time service has none, as without Jira, couldn't give one, or isn't there. */
  week?: Week;
  /** How asking the time service for the week went; absent without one to ask, or from a home older than the field. */
  weekState?: State;
}

/** The error fetchHome fails with when the gateway answers 401: the user has no session. */
export class SignedOutError extends Error {
  override name = 'SignedOutError';
}

/**
 * Fetches the Home; fails when the gateway can't be reached or answers with an error, and with a
 * SignedOutError when the user has no session.
 */
export async function fetchHome(): Promise<Home> {
  const response = await fetch('/api/home');
  if (response.status === 401) {
    throw new SignedOutError('GET /api/home: 401');
  }
  if (!response.ok) {
    throw new Error(`GET /api/home: ${response.status}`);
  }
  return response.json();
}
