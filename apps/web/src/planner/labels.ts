import type { Author, Conversation } from './conversation';
import type { StartStatus } from './prompt.svelte';
import type { SendStatus } from './reply.svelte';

/** The most characters of a plan's title that its row shows. */
const titleLength = 80;

/**
 * A plan's title as its row shows it, on one line: "Untitled plan" for a blank one, and a long one, such as a whole
 * prompt, cut to 80 characters, at a space if one is in the second half, and ended with "…".
 */
export function titleLabel(title: string): string {
  const line = title.replaceAll(/\s+/g, ' ').trim();
  if (line === '') {
    return 'Untitled plan';
  }
  // Characters, not UTF-16 code units, so the cut never splits an emoji's surrogate pair.
  const characters = [...line];
  if (characters.length <= titleLength) {
    return line;
  }
  // One character more than is shown, so a space at its end says that the shown ones end on a whole word.
  const cut = characters.slice(0, titleLength + 1).join('');
  const space = cut.lastIndexOf(' ');
  return `${space > cut.length / 2 ? cut.slice(0, space) : characters.slice(0, titleLength).join('')}…`;
}

/** The prompt's status line: "Starting…", "Started", why the start failed, or nothing before the first. */
export function startLabel(status: StartStatus): string {
  switch (status.kind) {
    case 'starting':
      return 'Starting…';
    case 'started':
      return 'Started';
    case 'failed':
      return status.reason;
    default:
      return '';
  }
}

/** A reply's or an answer's status line: "Sending…", "Sent", why the send failed, or nothing before the first. */
export function sendLabel(status: SendStatus): string {
  switch (status.kind) {
    case 'sending':
      return 'Sending…';
    case 'sent':
      return 'Sent';
    case 'failed':
      return status.reason;
    default:
      return '';
  }
}

/** Who wrote a message, as its row says: "You" or "Copilot". */
export function authorLabel(from: Author): string {
  return from === 'user' ? 'You' : 'Copilot';
}

/**
 * A plan's status line: "Copilot waits for your answer." while one of its questions waits, "Copilot is working…", why
 * the agent stopped, or nothing once it stopped without an error.
 */
export function workLabel(talk: Conversation): string {
  if (talk.questions.length > 0) {
    return 'Copilot waits for your answer.';
  }
  if (talk.working) {
    return 'Copilot is working…';
  }
  return talk.failure === '' ? '' : `Copilot stopped: ${talk.failure}`;
}
