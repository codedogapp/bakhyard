/** Who wrote a message of a plan's conversation. */
export type Author = 'user' | 'assistant';

/** A message of a plan's conversation, or for a delta the message's ID and its new text (decision 407). */
interface Message {
  id: string;
  from: Author;
  text: string;
}

/** Whether a plan's agent is working, and the error that stopped it, or "" (decision 702). */
interface Status {
  working: boolean;
  failure: string;
}

/** A question of the agent's that waits for the user's answer (decision 410). */
export interface Question {
  id: string;
  text: string;
  /** The answers that the agent offers, if any. */
  choices: string[];
  /** Whether the question takes an answer other than its choices, as it always does without any. */
  freeform: boolean;
}

/**
 * A plan's conversation as its page shows it: the whole messages, those the agent is still writing, the questions that
 * wait, in the order asked, and its status.
 */
export interface Conversation extends Status {
  messages: Message[];
  writing: Message[];
  questions: Question[];
}

/** An event of a plan's stream that changes its conversation, by its name, with its data. */
export type Change =
  | { name: 'conversation'; data: Conversation }
  | { name: 'message' | 'delta'; data: Message }
  | { name: 'question' | 'answered'; data: Question }
  | { name: 'status'; data: Status };

/** The names of the stream's events that change the conversation; the page leaves the plan to a later step. */
export const changeNames: readonly Change['name'][] = [
  'conversation',
  'message',
  'delta',
  'question',
  'answered',
  'status',
];

/**
 * The conversation after the change: a new conversation replaces it; a whole message joins the messages and leaves
 * those being written; a delta adds its text to the message being written with its ID, or starts one; a question
 * joins those that wait, and one answered leaves them; a status replaces the status. Once the agent has stopped, the
 * conversation keeps neither the messages being written nor the questions, as kept says.
 */
export function changed(talk: Conversation, change: Change): Conversation {
  switch (change.name) {
    case 'conversation':
      return kept(change.data);
    case 'message':
      return {
        ...talk,
        messages: [...talk.messages, change.data],
        writing: talk.writing.filter((message) => message.id !== change.data.id),
      };
    case 'delta':
      return { ...talk, writing: written(talk.writing, change.data) };
    case 'question':
      return { ...talk, questions: [...talk.questions, change.data] };
    case 'answered':
      return { ...talk, questions: talk.questions.filter((question) => question.id !== change.data.id) };
    case 'status':
      return kept({ ...talk, ...change.data });
  }
}

/**
 * The conversation without what ends with the agent's turn once the agent has stopped: the messages it was writing,
 * which it won't finish, as the service says, and the questions, which it took back, since a question that waits
 * holds the turn (decision 410). Such a question stays with the service, so it doesn't come back with the next turn.
 */
function kept(talk: Conversation): Conversation {
  return talk.working ? talk : { ...talk, writing: [], questions: [] };
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
