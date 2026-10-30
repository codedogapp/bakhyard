import { startPlan } from './api';

/** How the prompt's last start went: none yet, starting, started, or failed, with the reason to show. */
export type StartStatus = { kind: 'idle' | 'starting' | 'started' } | { kind: 'failed'; reason: string };

/** The Planner page's prompt for a new plan: its text as typed, and how its last start went. */
export class PromptState {
  text = $state('');
  status = $state.raw<StartStatus>({ kind: 'idle' });

  get busy(): boolean {
    return this.status.kind === 'starting';
  }

  /**
   * Starts a plan from the prompt, then empties it and calls started with the new plan's ID; a failed start keeps it.
   * Starts nothing for a blank prompt, or while a start runs.
   */
  async start(started: (id: string) => void): Promise<void> {
    if (this.busy || this.text.trim() === '') {
      return;
    }
    this.status = { kind: 'starting' };
    const result = await startPlan(this.text);
    if (result.kind === 'failed') {
      this.status = result;
      return;
    }
    this.text = '';
    this.status = { kind: 'started' };
    started(result.id);
  }
}
