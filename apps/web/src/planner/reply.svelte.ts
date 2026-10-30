import { sendMessage } from './api';

/** How a reply's or an answer's last send went: none yet, sending, sent, or failed, with the reason to show. */
export type SendStatus = { kind: 'idle' | 'sending' | 'sent' } | { kind: 'failed'; reason: string };

/** A plan's reply: its text as typed, and how its last send went. */
export class ReplyState {
  text = $state('');
  status = $state.raw<SendStatus>({ kind: 'idle' });

  get busy(): boolean {
    return this.status.kind === 'sending';
  }

  /**
   * Sends the reply to the plan with the ID, then empties it; a failed send keeps it. Sends nothing for a blank reply,
   * or while a send runs.
   */
  async send(id: string): Promise<void> {
    if (this.busy || this.text.trim() === '') {
      return;
    }
    this.status = { kind: 'sending' };
    const failure = await sendMessage(id, this.text);
    if (failure !== undefined) {
      this.status = { kind: 'failed', reason: failure };
      return;
    }
    this.text = '';
    this.status = { kind: 'sent' };
  }
}
