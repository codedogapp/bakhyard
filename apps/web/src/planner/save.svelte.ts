import { type Save, savePlan } from './api';

/** How a plan's last save went: none yet, saving, saved with the note's path, or failed with the reason to show. */
export type SaveStatus = { kind: 'idle' | 'saving' } | Save;

/** A plan's saves in the vault: how the last one went. */
export class SaveState {
  status = $state.raw<SaveStatus>({ kind: 'idle' });

  get busy(): boolean {
    return this.status.kind === 'saving';
  }

  /** Saves the plan of the plan with the ID, as its page's path holds it, in the vault, unless a save runs. */
  async save(id: string): Promise<void> {
    if (this.busy) {
      return;
    }
    this.status = { kind: 'saving' };
    this.status = await savePlan(id);
  }
}
