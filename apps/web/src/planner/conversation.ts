/** Who wrote a message of a plan's conversation. */
export type Author = 'user' | 'assistant';

/** A message of a plan's conversation, or for a delta the message's ID and its new text (decision 407). */
interface Message {
  id: string;
  from: Author;
  text: string;
}

/** Whether a plan's agent is working, and the error that stopped it, or "" (decision 702). */
export interface Status {
  working: boolean;
  failure: string;
}

/** A plan's conversation as its page shows it: the whole messages, those the agent is still writing, its status. */
export interface Conversation extends Status {
  messages: Message[];
  writing: Message[];
}

/** An event of a plan's stream that changes its conversation, by its name, with its data. */
export type Change =
  | { name: 'conversation'; data: Conversation }
  | { name: 'message' | 'delta'; data: Message }
  | { name: 'status'; data: Status };

/** The names of the stream's events that change the conversation; the page leaves the others to later steps. */
export const changeNames: readonly Change['name'][] = ['conversation', 'message', 'delta', 'status'];

/**
 * The conversation after the change: a new conversation replaces it; a whole message joins the messages and leaves
 * those being written; a delta adds its text to the message being written with its ID, or starts one; a status
 * replaces the status, and drops the messages being written once the agent stops, as the service does.
 */
export function changed(talk: Conversation, change: Change): Conversation {
  switch (change.name) {
    case 'conversation':
      return change.data;
    case 'message':
      return {
        ...talk,
        messages: [...talk.messages, change.data],
        writing: talk.writing.filter((message) => message.id !== change.data.id),
      };
    case 'delta':
      return { ...talk, writing: written(talk.writing, change.data) };
    case 'status':
      return { ...talk, ...change.data, writing: change.data.working ? talk.writing : [] };
  }
}

// The messages being written, with the delta's text added to the one with its ID, or as a new one.
function written(writing: Message[], delta: Message): Message[] {
  if (!writing.some((message) => message.id === delta.id)) {
    return [...writing, delta];
  }
  return writing.map((message) =>
    message.id === delta.id ? { ...message, text: message.text + delta.text } : message
  );
}
