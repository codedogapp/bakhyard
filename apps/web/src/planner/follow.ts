import { type Conversation, changed, changeNames } from './conversation';

/** What a plan's page shows: nothing while its stream opens, that it couldn't open, or the conversation. */
export type Chat = { kind: 'loading' | 'failed' } | { kind: 'loaded'; talk: Conversation };

/** How long a plan's page waits to open its stream again once the browser has given up on it. */
export const reopenMs = 5_000;

const empty: Conversation = { messages: [], writing: [], questions: [], working: false, failure: '' };

/**
 * Follows the stream of the plan id and calls show with what its page shows: the conversation as it changes, or failed
 * when the stream doesn't open. A stream that breaks once the conversation has come opens again, by itself or, once
 * the browser gives up on it, such as on a 502 while the planner restarts, reopenMs later; the page keeps what it
 * shows until the new conversation comes. Returns what stops it.
 */
export function followPlan(id: string, show: (chat: Chat) => void): () => void {
  let talk: Conversation | undefined;
  let events: EventSource;
  let timer: ReturnType<typeof setTimeout> | undefined;

  const open = () => {
    events = new EventSource(`/api/planner/plans/${id}/events`);
    for (const name of changeNames) {
      events.addEventListener(name, (event) => {
        // The stream's first event is the conversation, so a change never comes without one.
        talk = changed(talk ?? empty, { name, data: JSON.parse(event.data) });
        show({ kind: 'loaded', talk });
      });
    }
    events.addEventListener('error', () => {
      if (events.readyState !== EventSource.CLOSED) {
        return;
      }
      if (talk === undefined) {
        show({ kind: 'failed' });
        return;
      }
      timer = setTimeout(open, reopenMs);
    });
  };

  open();
  return () => {
    events.close();
    clearTimeout(timer);
  };
}
