/** A question that an agent asks and waits on, as the nudge service lists it. */
export interface Question {
  id: string;
  /** What the agent works on: its folder and its machine. */
  project: string;
  text: string;
  details: string;
  /** The answers to pick with one tap; none when the agent wants a typed reply. */
  options: string[];
  asked_at: string;
}

/** An agent's session and the note that says how its work goes. */
export interface Session {
  id: string;
  project: string;
  note: string;
  /** When the note was set; the zero time until the first. */
  noted_at: string;
}

/** The body of GET /api/nudge/state. */
export interface State {
  questions: Question[];
  sessions: Session[];
}

/** An answer: one of the question's options, a text, or both. */
export interface Answer {
  option?: string;
  text?: string;
}

/** The page's state: loading, failed to load, or loaded. */
export type Loaded = { kind: 'loading' | 'failed' } | { kind: 'loaded'; state: State };

/** Loads the pending questions and the sessions; never fails, since a failed load is one of the outcomes. */
export async function loadState(): Promise<Loaded> {
  try {
    const response = await fetch('/api/nudge/state');
    if (response.status !== 200) {
      return { kind: 'failed' };
    }
    return { kind: 'loaded', state: await response.json() };
  } catch {
    return { kind: 'failed' };
  }
}

/** Sends the answer to the question; the failure is the reason to show the user, or none when it went through. */
export async function sendAnswer(id: string, answer: Answer): Promise<string | undefined> {
  try {
    const response = await fetch(`/api/nudge/questions/${encodeURIComponent(id)}/answer`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(answer),
    });
    if (response.status === 204) {
      return undefined;
    }
    if (response.status === 404) {
      return 'The agent no longer waits for this question.';
    }
    return (await response.text()).trim() || `The answer failed (${response.status}).`;
  } catch {
    return "The answer didn't go through: nudge isn't reachable.";
  }
}
