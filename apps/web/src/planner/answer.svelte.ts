import { sendAnswer } from './api';
import type { SendStatus } from './reply.svelte';

/** The answer to one of a plan's questions: the text typed for it, and how its last send went. */
export class AnswerState {
  text = $state('');
  status = $state.raw<SendStatus>({ kind: 'idle' });

  get busy(): boolean {
    return this.status.kind === 'sending';
  }

  /**
   * Sends the choice, or else the text as typed, as the answer to the question of the plan with the ID. Sends nothing
   * for a blank answer, or while a send runs. The row goes once the stream says the question has its answer, so a
   * sent text stays until then, as a failed one does.
   */
  async send(id: string, question: string, choice?: string): Promise<void> {
    const answer = choice ?? this.text;
    if (this.busy || answer.trim() === '') {
      return;
    }
    this.status = { kind: 'sending' };
    const failure = await sendAnswer(id, question, answer);
    this.status = failure === undefined ? { kind: 'sent' } : { kind: 'failed', reason: failure };
  }
}
