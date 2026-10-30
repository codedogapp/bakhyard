/** A plan, as the planner service lists it (decision 413). */
export interface Plan {
  id: string;
  /** The CLI's name for the plan, which may be its prompt, or "" before it has one. */
  title: string;
  started_at: string;
  /** When the CLI's record of the plan last changed. */
  updated_at: string;
}

/** The Planner page's list: loading, off when Copilot isn't set up, failed to load, or the plans. */
export type Plans = { kind: 'loading' | 'off' | 'failed' } | { kind: 'loaded'; plans: Plan[] };

/**
 * Loads the plans, the last updated first. A 404 says that Copilot isn't set up: the planner's own, or the gateway's
 * while it doesn't forward to the planner. Never fails, since a failed load is one of the outcomes.
 */
export async function loadPlans(): Promise<Plans> {
  try {
    const response = await fetch('/api/planner/plans');
    if (response.status === 404) {
      return { kind: 'off' };
    }
    if (response.status !== 200) {
      return { kind: 'failed' };
    }
    const body: { plans: Plan[] } = await response.json();
    return { kind: 'loaded', plans: body.plans };
  } catch {
    return { kind: 'failed' };
  }
}
